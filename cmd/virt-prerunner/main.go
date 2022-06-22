package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/docker/libnetwork/resolvconf"
	"github.com/docker/libnetwork/types"
	"github.com/subgraph/libmacouflage"
	"github.com/vishvananda/netlink"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
	"github.com/smartxworks/virtink/pkg/cloudhypervisor"
)

func main() {
	vmJSON, err := base64.StdEncoding.DecodeString(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to decode VM JSON: %s", err)
	}

	var vm virtv1alpha1.VirtualMachine
	if err := json.Unmarshal(vmJSON, &vm); err != nil {
		log.Fatalf("Failed to unmarshal VM: %s", err)
	}

	vmConfig, err := buildVMConfig(context.Background(), &vm)
	if err != nil {
		log.Fatalf("Failed to build VM config: %s", err)
	}

	cloudHypervisorCmd := []string{"cloud-hypervisor", "--api-socket", "/var/run/virtink/ch.sock", "--console", "pty", "--serial", "tty"}
	cloudHypervisorCmd = append(cloudHypervisorCmd, "--kernel", vmConfig.Kernel.Path)
	if vmConfig.Cmdline != nil {
		cloudHypervisorCmd = append(cloudHypervisorCmd, "--cmdline", fmt.Sprintf("'%s'", vmConfig.Cmdline.Args))
	}
	cloudHypervisorCmd = append(cloudHypervisorCmd, "--cpus", fmt.Sprintf("boot=%d,topology=%d:%d:%d:%d",
		vmConfig.Cpus.BootVcpus, vmConfig.Cpus.Topology.ThreadsPerCore, vmConfig.Cpus.Topology.CoresPerDie,
		vmConfig.Cpus.Topology.DiesPerPackage, vmConfig.Cpus.Topology.Packages))
	cloudHypervisorCmd = append(cloudHypervisorCmd, "--memory", fmt.Sprintf("size=%d", vmConfig.Memory.Size))

	if len(vmConfig.Disks) > 0 {
		cloudHypervisorCmd = append(cloudHypervisorCmd, "--disk")
		for _, disk := range vmConfig.Disks {
			arg := fmt.Sprintf("id=%s,path=%s", disk.Id, disk.Path)
			if disk.Readonly {
				arg = arg + ",readonly=on"
			}
			cloudHypervisorCmd = append(cloudHypervisorCmd, arg)
		}
	}

	if len(vmConfig.Net) > 0 {
		cloudHypervisorCmd = append(cloudHypervisorCmd, "--net")
		for _, net := range vmConfig.Net {
			cloudHypervisorCmd = append(cloudHypervisorCmd, fmt.Sprintf("id=%s,mac=%s,tap=%s", net.Id, net.Mac, net.Tap))
		}
	}

	fmt.Println(strings.Join(cloudHypervisorCmd, " "))
}

func buildVMConfig(ctx context.Context, vm *virtv1alpha1.VirtualMachine) (*cloudhypervisor.VmConfig, error) {
	vmConfig := cloudhypervisor.VmConfig{
		Kernel: &cloudhypervisor.KernelConfig{
			Path: "/var/lib/cloud-hypervisor/hypervisor-fw",
		},
		Cpus: &cloudhypervisor.CpusConfig{
			BootVcpus: int(vm.Spec.Instance.CPU.Sockets * vm.Spec.Instance.CPU.CoresPerSocket),
			Topology: &cloudhypervisor.CpuTopology{
				Packages:       int(vm.Spec.Instance.CPU.Sockets),
				DiesPerPackage: 1,
				CoresPerDie:    int(vm.Spec.Instance.CPU.CoresPerSocket),
				ThreadsPerCore: 1,
			},
		},
		Memory: &cloudhypervisor.MemoryConfig{
			Size: vm.Spec.Instance.Memory.Size.Value(),
		},
	}

	if vm.Spec.Instance.Kernel != nil {
		vmConfig.Kernel.Path = "/mnt/virtink-kernel/vmlinux"
		vmConfig.Cmdline = &cloudhypervisor.CmdLineConfig{
			Args: vm.Spec.Instance.Kernel.Cmdline,
		}
	}

	for _, disk := range vm.Spec.Instance.Disks {
		for _, volume := range vm.Spec.Volumes {
			if volume.Name == disk.Name {
				diskConfig := cloudhypervisor.DiskConfig{
					Id: disk.Name,
				}
				switch {
				case volume.ContainerDisk != nil:
					diskConfig.Path = fmt.Sprintf("/mnt/%s/disk.raw", volume.Name)
				case volume.CloudInit != nil:
					diskConfig.Path = fmt.Sprintf("/mnt/%s/cloud-init.iso", volume.Name)
				case volume.ContainerRootfs != nil:
					diskConfig.Path = fmt.Sprintf("/mnt/%s/rootfs.raw", volume.Name)
				default:
					return nil, fmt.Errorf("invalid source of volume %q", volume.Name)
				}

				if disk.ReadOnly != nil && *disk.ReadOnly {
					diskConfig.Readonly = true
				}

				vmConfig.Disks = append(vmConfig.Disks, &diskConfig)
				break
			}
		}
	}

	for _, iface := range vm.Spec.Instance.Interfaces {
		for networkIndex, network := range vm.Spec.Networks {
			if network.Name == iface.Name {
				netConfig := cloudhypervisor.NetConfig{
					Id: iface.Name,
				}

				var linkName string
				switch {
				case network.Pod != nil:
					linkName = "eth0"
				case network.Multus != nil:
					linkName = fmt.Sprintf("net%d", networkIndex)
				default:
					return nil, fmt.Errorf("invalid source of network %q", network.Name)
				}

				if err := setupBridgeNetwork(linkName, fmt.Sprintf("169.254.%d.1/30", 200+networkIndex), &netConfig); err != nil {
					return nil, fmt.Errorf("setup bridge network: %s", err)
				}

				vmConfig.Net = append(vmConfig.Net, &netConfig)
				break
			}
		}
	}

	return &vmConfig, nil
}

func setupBridgeNetwork(linkName string, cidr string, netConfig *cloudhypervisor.NetConfig) error {
	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("parse CIDR: %s", err)
	}

	bridgeIP, err := nextIP(subnet.IP, subnet)
	if err != nil {
		return fmt.Errorf("generate bridge IP: %s", err)
	}
	bridgeIPNet := net.IPNet{
		IP:   bridgeIP,
		Mask: subnet.Mask,
	}

	bridgeName := fmt.Sprintf("br-%s", linkName)
	bridge, err := createBridge(bridgeName, &bridgeIPNet)
	if err != nil {
		return fmt.Errorf("create bridge: %s", err)
	}

	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return fmt.Errorf("get link: %s", err)
	}
	linkMAC := link.Attrs().HardwareAddr
	netConfig.Mac = linkMAC.String()

	var linkAddr *net.IPNet
	linkAddrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("list link addrs: %s", err)
	}
	if len(linkAddrs) > 0 {
		linkAddr = linkAddrs[0].IPNet
	}

	var linkGateway net.IP
	linkRoutes, err := netlink.RouteList(link, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("list link routes: %s", err)
	}
	if len(linkRoutes) > 0 {
		linkGateway = linkRoutes[0].Gw
	}

	if err := netlink.LinkSetDown(link); err != nil {
		return fmt.Errorf("down link: %s", err)
	}

	if _, err := libmacouflage.SpoofMacSameVendor(linkName, false); err != nil {
		return fmt.Errorf("spoof link MAC: %s", err)
	}

	if linkAddr != nil {
		if err := netlink.AddrDel(link, &linkAddrs[0]); err != nil {
			return fmt.Errorf("delete link address: %s", err)
		}

		originalLinkName := link.Attrs().Name
		newLinkName := fmt.Sprintf("%s-nic", originalLinkName)

		if err := netlink.LinkSetName(link, newLinkName); err != nil {
			return fmt.Errorf("rename link: %s", err)
		}

		dummy := &netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				Name: originalLinkName,
			},
		}
		if err := netlink.LinkAdd(dummy); err != nil {
			return fmt.Errorf("add dummy interface: %s", err)
		}
		if err := netlink.AddrReplace(dummy, &linkAddrs[0]); err != nil {
			return fmt.Errorf("replace dummy interface address: %s", err)
		}
	}

	if err := netlink.LinkSetMaster(link, bridge); err != nil {
		return fmt.Errorf("add link to bridge: %s", err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("up link: %s", err)
	}

	tapName := fmt.Sprintf("tap-%s", linkName)
	if _, err := createTap(bridge, tapName); err != nil {
		return fmt.Errorf("create tap: %s", err)
	}
	netConfig.Tap = tapName

	if linkAddr != nil {
		if err := startDHCPServer(bridgeName, linkMAC, linkAddr, linkGateway); err != nil {
			return fmt.Errorf("start DHCP server: %s", err)
		}
	}
	return nil
}

func nextIP(ip net.IP, subnet *net.IPNet) (net.IP, error) {
	nextIP := make(net.IP, len(ip))
	copy(nextIP, ip)
	for j := len(nextIP) - 1; j >= 0; j-- {
		nextIP[j]++
		if nextIP[j] > 0 {
			break
		}
	}
	if subnet != nil && !subnet.Contains(nextIP) {
		return nil, fmt.Errorf("no more available IP in subnet %q", subnet.String())
	}
	return nextIP, nil
}

func createBridge(bridgeName string, bridgeIPNet *net.IPNet) (netlink.Link, error) {
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: bridgeName,
		},
	}
	if err := netlink.LinkAdd(bridge); err != nil {
		return nil, err
	}

	if err := netlink.AddrAdd(bridge, &netlink.Addr{IPNet: bridgeIPNet}); err != nil {
		return nil, fmt.Errorf("set bridge addr: %s", err)
	}

	if err := netlink.LinkSetUp(bridge); err != nil {
		return nil, fmt.Errorf("up bridge: %s", err)
	}
	return bridge, nil
}

func createTap(bridge netlink.Link, tapName string) (netlink.Link, error) {
	tap := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{
			Name: tapName,
		},
		Mode:  netlink.TUNTAP_MODE_TAP,
		Flags: netlink.TUNTAP_DEFAULTS,
	}
	if err := netlink.LinkAdd(tap); err != nil {
		return nil, err
	}

	if err := netlink.LinkSetMaster(tap, bridge); err != nil {
		return nil, fmt.Errorf("add tap to bridge: %s", err)
	}

	if err := netlink.LinkSetUp(tap); err != nil {
		return nil, fmt.Errorf("up tap: %s", err)
	}
	return tap, nil
}

//go:embed dnsmasq.conf
var dnsmasqConf string

func startDHCPServer(ifaceName string, mac net.HardwareAddr, ipNet *net.IPNet, gateway net.IP) error {
	rc, err := resolvconf.Get()
	if err != nil {
		return fmt.Errorf("get resolvconf: %s", err)
	}

	dnsmasqPIDPath := fmt.Sprintf("/var/run/virtink/dnsmasq/%s.pid", ifaceName)
	if err := os.MkdirAll(filepath.Dir(dnsmasqPIDPath), 0755); err != nil {
		return fmt.Errorf("create dnsmasq PID dir: %s", err)
	}

	dnsmasqConfPath := fmt.Sprintf("/var/run/virtink/dnsmasq/%s.conf", ifaceName)
	if err := os.MkdirAll(filepath.Dir(dnsmasqConfPath), 0755); err != nil {
		return fmt.Errorf("create dnsmasq config dir: %s", err)
	}

	dnsmasqConfFile, err := os.Create(dnsmasqConfPath)
	if err != nil {
		return fmt.Errorf("create dnsmasq config file: %s", err)
	}
	defer dnsmasqConfFile.Close()

	data := map[string]string{
		"iface":        ifaceName,
		"mac":          mac.String(),
		"ip":           ipNet.IP.String(),
		"mask":         net.IP(ipNet.Mask).String(),
		"gateway":      gateway.String(),
		"dnsServer":    strings.Join(resolvconf.GetNameservers(rc.Content, types.IPv4), ","),
		"domainSearch": strings.Join(resolvconf.GetSearchDomains(rc.Content), ","),
	}
	if err := template.Must(template.New("dnsmasq.conf").Parse(dnsmasqConf)).Execute(dnsmasqConfFile, data); err != nil {
		return fmt.Errorf("write dnsmasq config file: %s", err)
	}

	if _, err := executeCommand("dnsmasq", fmt.Sprintf("--conf-file=%s", dnsmasqConfPath), fmt.Sprintf("--pid-file=%s", dnsmasqPIDPath)); err != nil {
		return fmt.Errorf("start dnsmasq: %s", err)
	}
	return nil
}

func executeCommand(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%q: %s: %s", cmd.String(), err, output)
	}
	return string(output), nil
}

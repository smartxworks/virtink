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
	"runtime"
	"strings"
	"syscall"
	"text/template"

	"github.com/docker/libnetwork/resolvconf"
	"github.com/docker/libnetwork/types"
	userspacecni "github.com/intel/userspace-cni-network-plugin/pkg/types"
	netv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/namsral/flag"
	"github.com/subgraph/libmacouflage"
	"github.com/vishvananda/netlink"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
	"github.com/smartxworks/virtink/pkg/cloudhypervisor"
	"github.com/smartxworks/virtink/pkg/cpuset"
	"github.com/smartxworks/virtink/pkg/sanlock"
)

func main() {
	var vmData string
	var receiveMigration bool
	flag.StringVar(&vmData, "vm-data", vmData, "Base64 encoded VM json data")
	flag.BoolVar(&receiveMigration, "receive-migration", receiveMigration, "Receive migration instead of starting a new VM")
	flag.Parse()

	vmJSON, err := base64.StdEncoding.DecodeString(vmData)
	if err != nil {
		log.Fatalf("Failed to decode VM data: %s", err)
	}

	var vm virtv1alpha1.VirtualMachine
	if err := json.Unmarshal(vmJSON, &vm); err != nil {
		log.Fatalf("Failed to unmarshal VM: %s", err)
	}

	if len(vm.Spec.Locks) > 0 {
		resources := os.Getenv("LOCKSPACE_RESOURCE")
		if err := sanlock.AcquireResourceLease(strings.Split(resources, " "), vm.Name); err != nil {
			log.Fatalf("Failed to acquire lock: %s", err)
		}
	}

	vmConfig, err := buildVMConfig(context.Background(), &vm)
	if err != nil {
		log.Fatalf("Failed to build VM config: %s", err)
	}

	if !receiveMigration {
		vmConfigFile, err := os.Create("/var/run/virtink/vm-config.json")
		if err != nil {
			log.Fatalf("Failed to create VM config file: %s", err)
		}

		if err := json.NewEncoder(vmConfigFile).Encode(vmConfig); err != nil {
			log.Fatalf("Failed to write VM config to file: %s", err)
		}
		vmConfigFile.Close()

		log.Println("Succeeded to setup")
	}

	if err := syscall.Exec("/usr/bin/cloud-hypervisor", []string{"cloud-hypervisor", "--api-socket", "/var/run/virtink/ch.sock"}, nil); err != nil {
		log.Fatalf("failed to start cloud-hypervisor: %s", err)
	}
}

func buildVMConfig(ctx context.Context, vm *virtv1alpha1.VirtualMachine) (*cloudhypervisor.VmConfig, error) {
	vmConfig := cloudhypervisor.VmConfig{
		Console: &cloudhypervisor.ConsoleConfig{
			Mode: "Pty",
		},
		Serial: &cloudhypervisor.ConsoleConfig{
			Mode: "Tty",
		},
		Payload: &cloudhypervisor.PayloadConfig{
			Kernel: "/var/lib/cloud-hypervisor/hypervisor-fw",
		},
		Cpus: &cloudhypervisor.CpusConfig{
			MaxVcpus:  int(vm.Spec.Instance.CPU.Sockets * vm.Spec.Instance.CPU.CoresPerSocket),
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

	if runtime.GOARCH == "arm64" {
		vmConfig.Payload.Kernel = "/var/lib/cloud-hypervisor/CLOUDHV_EFI.fd"
	}

	if vm.Spec.Instance.Kernel != nil {
		vmConfig.Payload.Kernel = "/mnt/virtink-kernel/vmlinux"
		vmConfig.Payload.Cmdline = vm.Spec.Instance.Kernel.Cmdline
	}

	if vm.Spec.Instance.CPU.DedicatedCPUPlacement {
		cpuSet, err := cpuset.Get()
		if err != nil {
			return nil, fmt.Errorf("get CPU set: %s", err)
		}

		pcpus := cpuSet.ToSlice()
		numVCPUs := int(vm.Spec.Instance.CPU.Sockets * vm.Spec.Instance.CPU.CoresPerSocket)
		if len(pcpus) != numVCPUs {
			// TODO: report an event to object VM
			return nil, fmt.Errorf("number of pCPUs and vCPUs must match")
		}

		for i := 0; i < numVCPUs; i++ {
			vmConfig.Cpus.Affinity = append(vmConfig.Cpus.Affinity, &cloudhypervisor.CpuAffinity{
				Vcpu:     i,
				HostCpus: []int{pcpus[i]},
			})
		}
	}

	if vm.Spec.Instance.Memory.Hugepages != nil {
		vmConfig.Memory.Hugepages = true
	}

	blockVolumes := map[string]bool{}
	for _, volume := range strings.Split(os.Getenv("BLOCK_VOLUMES"), ",") {
		blockVolumes[volume] = true
	}

	for _, disk := range vm.Spec.Instance.Disks {
		for _, volume := range vm.Spec.Volumes {
			if volume.Name == disk.Name {
				diskConfig := cloudhypervisor.DiskConfig{
					Id:     disk.Name,
					Direct: true,
				}
				switch {
				case volume.ContainerDisk != nil:
					diskConfig.Path = fmt.Sprintf("/mnt/%s/disk.raw", volume.Name)
				case volume.CloudInit != nil:
					diskConfig.Path = fmt.Sprintf("/mnt/%s/cloud-init.iso", volume.Name)
				case volume.ContainerRootfs != nil:
					diskConfig.Path = fmt.Sprintf("/mnt/%s/rootfs.raw", volume.Name)
				case volume.PersistentVolumeClaim != nil, volume.DataVolume != nil:
					if blockVolumes[volume.Name] {
						if volume.IsHotpluggable() {
							diskConfig.Path = fmt.Sprintf("/hotplug-volumes/%s", volume.Name)
						} else {
							diskConfig.Path = fmt.Sprintf("/mnt/%s", volume.Name)
						}
					} else {
						if volume.IsHotpluggable() {
							diskConfig.Path = filepath.Join("/hotplug-volumes", fmt.Sprintf("%s.img", volume.Name))
						} else {
							diskConfig.Path = filepath.Join("/mnt", volume.Name, "disk.img")
						}
					}
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

	for _, fs := range vm.Spec.Instance.FileSystems {
		vmConfig.Memory.Shared = true

		if err := os.MkdirAll("/var/run/virtink/virtiofsd", 0755); err != nil {
			return nil, fmt.Errorf("create virtiofsd socket dir: %s", err)
		}

		for _, volume := range vm.Spec.Volumes {
			if volume.Name == fs.Name {
				socketPath := fmt.Sprintf("/var/run/virtink/virtiofsd/%s.sock", volume.Name)
				if err := exec.Command("/usr/lib/qemu/virtiofsd", "--socket-path="+socketPath, "-o", "source=/mnt/"+volume.Name, "-o", "sandbox=chroot").Start(); err != nil {
					return nil, fmt.Errorf("start virtiofsd: %s", err)
				}

				fsConfig := cloudhypervisor.FsConfig{
					Id:        fs.Name,
					Socket:    socketPath,
					Tag:       fs.Name,
					NumQueues: 1,
					QueueSize: 1024,
				}
				vmConfig.Fs = append(vmConfig.Fs, &fsConfig)
				break
			}
		}
	}

	networkStatusList := []netv1.NetworkStatus{}
	if os.Getenv("NETWORK_STATUS") != "" {
		if err := json.Unmarshal([]byte(os.Getenv("NETWORK_STATUS")), &networkStatusList); err != nil {
			return nil, err
		}
	}

	for _, iface := range vm.Spec.Instance.Interfaces {
		for networkIndex, network := range vm.Spec.Networks {
			if network.Name != iface.Name {
				continue
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

			switch {
			case iface.Bridge != nil:
				netConfig := cloudhypervisor.NetConfig{
					Id: iface.Name,
				}
				if err := setupBridgeNetwork(linkName, fmt.Sprintf("169.254.%d.1/30", 200+networkIndex), &netConfig); err != nil {
					return nil, fmt.Errorf("setup bridge network: %s", err)
				}
				vmConfig.Net = append(vmConfig.Net, &netConfig)
			case iface.Masquerade != nil:
				netConfig := cloudhypervisor.NetConfig{
					Id:  iface.Name,
					Mac: iface.MAC,
				}
				if err := setupMasqueradeNetwork(linkName, iface.Masquerade.CIDR, &netConfig); err != nil {
					return nil, fmt.Errorf("setup masquerade network: %s", err)
				}
				vmConfig.Net = append(vmConfig.Net, &netConfig)
			case iface.SRIOV != nil:
				for _, networkStatus := range networkStatusList {
					if networkStatus.Interface == linkName && networkStatus.DeviceInfo != nil && networkStatus.DeviceInfo.Pci != nil {
						sriovDeviceConfig := cloudhypervisor.DeviceConfig{
							Id:   iface.Name,
							Path: fmt.Sprintf("/sys/bus/pci/devices/%s", networkStatus.DeviceInfo.Pci.PciAddress),
						}
						vmConfig.Devices = append(vmConfig.Devices, &sriovDeviceConfig)
					}
				}
			case iface.VhostUser != nil:
				netConfig := cloudhypervisor.NetConfig{
					Id:        iface.Name,
					Mac:       iface.MAC,
					VhostUser: true,
					VhostMode: "Server",
				}
				if err := setupVhostUserNetwork(linkName, &netConfig); err != nil {
					return nil, fmt.Errorf("setup vhost-user network: %s", err)
				}
				vmConfig.Net = append(vmConfig.Net, &netConfig)
				vmConfig.Memory.Shared = true
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

	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return fmt.Errorf("get link: %s", err)
	}
	netConfig.Mtu = link.Attrs().MTU

	bridgeName := fmt.Sprintf("br-%s", linkName)
	bridge, err := createBridge(bridgeName, &bridgeIPNet, link.Attrs().MTU)
	if err != nil {
		return fmt.Errorf("create bridge: %s", err)
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

	linkRoutes, err := netlink.RouteList(link, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("list link routes: %s", err)
	}

	if err := netlink.LinkSetDown(link); err != nil {
		return fmt.Errorf("down link: %s", err)
	}

	if _, err := libmacouflage.SpoofMacSameVendor(linkName, false); err != nil {
		return fmt.Errorf("spoof link MAC: %s", err)
	}

	newLinkName := link.Attrs().Name
	if linkAddr != nil {
		if err := netlink.AddrDel(link, &linkAddrs[0]); err != nil {
			return fmt.Errorf("delete link address: %s", err)
		}

		originalLinkName := link.Attrs().Name
		newLinkName = fmt.Sprintf("%s-nic", originalLinkName)

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

	if _, err := executeCommand("bridge", "link", "set", "dev", newLinkName, "learning", "off"); err != nil {
		return fmt.Errorf("disable port MAC learning on bridge: %s", err)
	}

	tapName := fmt.Sprintf("tap-%s", linkName)
	if _, err := createTap(bridge, tapName, link.Attrs().MTU); err != nil {
		return fmt.Errorf("create tap: %s", err)
	}
	netConfig.Tap = tapName

	if linkAddr != nil {
		var linkGateway net.IP
		var routes []netlink.Route
		for _, route := range linkRoutes {
			if route.Dst == nil && len(route.Src) == 0 && len(route.Gw) == 0 {
				continue
			}
			if len(linkGateway) == 0 && route.Dst == nil {
				linkGateway = route.Gw
			}
			routes = append(routes, route)
		}
		if err := startDHCPServer(bridgeName, linkMAC, linkAddr, linkGateway, routes); err != nil {
			return fmt.Errorf("start DHCP server: %s", err)
		}
	}
	return nil
}

func setupMasqueradeNetwork(linkName string, cidr string, netConfig *cloudhypervisor.NetConfig) error {
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

	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return fmt.Errorf("get link: %s", err)
	}
	netConfig.Mtu = link.Attrs().MTU

	bridgeName := fmt.Sprintf("br-%s", linkName)
	bridge, err := createBridge(bridgeName, &bridgeIPNet, link.Attrs().MTU)
	if err != nil {
		return fmt.Errorf("create bridge: %s", err)
	}

	vmIP, err := nextIP(bridgeIP, subnet)
	if err != nil {
		return fmt.Errorf("generate vm IP: %s", err)
	}
	vmIPNet := &net.IPNet{
		IP:   vmIP,
		Mask: subnet.Mask,
	}

	if _, err := executeCommand("iptables", "-t", "nat", "-A", "POSTROUTING", "-o", linkName, "-j", "MASQUERADE"); err != nil {
		return fmt.Errorf("add masquerade rule: %s", err)
	}
	if _, err := executeCommand("iptables", "-t", "nat", "-A", "PREROUTING", "-i", linkName, "-j", "DNAT", "--to-destination", vmIP.String()); err != nil {
		return fmt.Errorf("add prerouting rule: %s", err)
	}

	tapName := fmt.Sprintf("tap-%s", linkName)
	if _, err := createTap(bridge, tapName, link.Attrs().MTU); err != nil {
		return fmt.Errorf("create tap: %s", err)
	}
	netConfig.Tap = tapName

	vmMAC, err := net.ParseMAC(netConfig.Mac)
	if err != nil {
		return fmt.Errorf("parse VM MAC: %s", err)
	}

	if err := startDHCPServer(bridgeName, vmMAC, vmIPNet, bridgeIP, nil); err != nil {
		return fmt.Errorf("start DHCP server: %s", err)
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

func createBridge(bridgeName string, bridgeIPNet *net.IPNet, mtu int) (netlink.Link, error) {
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: bridgeName,
			MTU:  mtu,
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

func createTap(bridge netlink.Link, tapName string, mtu int) (netlink.Link, error) {
	tap := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{
			Name: tapName,
			MTU:  mtu,
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

	createdTap, err := netlink.LinkByName(tapName)
	if err != nil {
		return nil, fmt.Errorf("get tap: %s", err)
	}
	return createdTap, nil
}

//go:embed dnsmasq.conf
var dnsmasqConf string

func startDHCPServer(ifaceName string, mac net.HardwareAddr, ipNet *net.IPNet, gateway net.IP, routes []netlink.Route) error {
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
		"routes":       sortAndFormatRoutes(routes),
		"dnsServer":    strings.Join(resolvconf.GetNameservers(rc.Content, types.IPv4), ","),
		"domainSearch": strings.Join(resolvconf.GetSearchDomains(rc.Content), ","),
	}

	if len(gateway) > 0 {
		data["gateway"] = gateway.String()
	}

	if err := template.Must(template.New("dnsmasq.conf").Parse(dnsmasqConf)).Execute(dnsmasqConfFile, data); err != nil {
		return fmt.Errorf("write dnsmasq config file: %s", err)
	}

	if _, err := executeCommand("dnsmasq", fmt.Sprintf("--conf-file=%s", dnsmasqConfPath), fmt.Sprintf("--pid-file=%s", dnsmasqPIDPath)); err != nil {
		return fmt.Errorf("start dnsmasq: %s", err)
	}
	return nil
}

func sortAndFormatRoutes(routes []netlink.Route) string {
	var sortedRoutes []netlink.Route
	var defaultRoutes []netlink.Route
	for _, route := range routes {
		if route.Dst == nil {
			defaultRoutes = append(defaultRoutes, route)
			continue
		}
		sortedRoutes = append(sortedRoutes, route)
	}
	sortedRoutes = append(sortedRoutes, defaultRoutes...)

	items := []string{}
	for _, route := range sortedRoutes {
		if len(route.Gw) == 0 {
			route.Gw = net.IPv4(0, 0, 0, 0)
		}
		if route.Dst == nil {
			route.Dst = &net.IPNet{
				IP:   net.IPv4(0, 0, 0, 0),
				Mask: net.CIDRMask(0, 32),
			}
		}
		items = append(items, route.Dst.String(), route.Gw.String())
	}
	return strings.Join(items, ",")
}

func setupVhostUserNetwork(linkName string, netConfig *cloudhypervisor.NetConfig) error {
	netType := os.Getenv("NET_TYPE")
	if netType == "" {
		return fmt.Errorf("network type not found")
	}

	var socketPath string
	mtu := 1500
	switch netType {
	case "kube-ovn":
		socketPath := os.Getenv("VHOST_USER_SOCKET")
		if socketPath == "" {
			return fmt.Errorf("vhost-user socket path not found")
		}
		link, err := netlink.LinkByName("eth0")
		if err != nil {
			return fmt.Errorf("get link: %s", err)
		}
		mtu = link.Attrs().MTU
	case "userspace":
		userspaceConfigData := os.Getenv("USERSPACE_CONFIGURATION_DATA")
		if userspaceConfigData == "" {
			return fmt.Errorf("userspace configuration data not found")
		}
		var configData []userspacecni.ConfigurationData
		if err := json.Unmarshal([]byte(userspaceConfigData), &configData); err != nil {
			return fmt.Errorf("unmarshal userspace configuration data: %s", err)
		}
		var containerID string
		var socketFile string
		for _, config := range configData {
			if config.IfName == linkName {
				containerID = config.ContainerId
				socketFile = config.Config.VhostConf.Socketfile
				break
			}
		}
		if containerID == "" || socketFile == "" {
			return fmt.Errorf("vhost-user link not found")
		}
		socketDir := fmt.Sprintf("/var/run/vhost-user/%s", containerID[0:12])
		if _, err := os.Stat(socketDir); err != nil {
			if os.IsNotExist(err) {
				socketPath = fmt.Sprintf("/var/run/vhost-user/%s", socketFile)
			} else {
				return err
			}
		} else {
			socketPath = fmt.Sprintf("%s/%s", socketDir, socketFile)
		}
	}

	netConfig.Mtu = mtu
	netConfig.VhostSocket = socketPath
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

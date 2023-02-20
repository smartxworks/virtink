package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os/exec"
	"time"

	"github.com/mdlayher/ndp"
	"golang.org/x/net/ipv6"
)

var (
	iface         string
	router        string
	isRemoteRoute bool
	client        string
	clientHWAddr  string
	prefix        string

	linkLocalAllRouters = netip.MustParseAddr("ff02::2")
)

func main() {
	log.SetPrefix("route-advertisement-daemon: ")
	src, dst, cidr, err := validateVars()
	if err != nil {
		log.Fatalf("ERROR: validate vars: %s", err)
	}
	if src != nil && isRemoteRoute {
		if _, err := executeCommand("ip", "-6", "neigh", "add", client, "lladdr", clientHWAddr, "dev", iface); err != nil {
			log.Fatalf("ERROR: add neighbor entry for client: %s", err)
		}

		ipv6LLA := src
		if !src.IsLinkLocalUnicast() {
			mac, err := tryDiscoverNeighborMAC(iface, src, 5)
			if err != nil {
				log.Fatalf("ERROR: discover router MAC: %s", err)
			}
			ipv6LLA = generateEUI64Address(net.ParseIP("fe80::0"), mac)
			mac2, err := tryDiscoverNeighborMAC(iface, ipv6LLA, 5)
			if err != nil {
				log.Fatalf("ERROR: discover router MAC: %s", err)
			}
			if mac.String() != mac2.String() {
				log.Fatalf("ERROR: failed to get router link-local address")
			}
		}

		if _, err := executeCommand("ip6tables", "-A", "OUTPUT", "-o", iface, "--src", ipv6LLA.String(), "-p", "icmpv6", "--icmpv6-type", "neighbor-solicitation", "-j", "DROP"); err != nil {
			log.Fatalf("ERROR: drop neighbor solicitation on interface: %s", err)
		}
		if _, err := executeCommand("ip6tables", "-A", "OUTPUT", "-o", iface, "--src", ipv6LLA.String(), "-p", "icmpv6", "--icmpv6-type", "neighbor-advertisement", "-j", "DROP"); err != nil {
			log.Fatalf("ERROR: drop neighbor advertisement on interface: %s", err)
		}

		// As described in RFC 4861 section-4.2, the srouce address of RA must be the link-local
		// address assigned to the interface from which the message is sent, so the LLA of default
		// router have to be added to the interface. And the followings need to be done.
		//  1.Disable DAD of the interface
		//  2.Add static neighbor entry for the client, otherwise the interface will send a NS
		//    message with it's MAC in options to client
		//  3.Drop NA message from interface responsed to NS message learning default router LLA
		//  4.Drop NS message from interface with default router LLA in options
		if executeCommand("ip", "addr", "add", fmt.Sprintf("%s/64", ipv6LLA.String()), "dev", iface); err != nil {
			log.Fatalf("ERROR: add IPv6 addr to the interface: %s", err)
		}

		src = ipv6LLA
	}

	if err := startRouteAdvertisement(iface, src, dst, cidr); err != nil {
		log.Fatalf("ERROR: start route advertisement: %s", err)
	}
}

func validateVars() (net.IP, net.IP, *net.IPNet, error) {
	if iface == "" {
		return nil, nil, nil, fmt.Errorf("the interface may not be empty")
	}

	var src net.IP
	if router != "" {
		src = net.ParseIP(router)
		if src == nil {
			return nil, nil, nil, fmt.Errorf("the router IPv6 address (%s) is illegal", router)
		}
		if isRemoteRoute {
			if clientHWAddr == "" {
				return nil, nil, nil, fmt.Errorf("the client-hardware-addr may not be empty when router is remote")
			}
		}
	}

	if client == "" {
		if clientHWAddr == "" {
			return nil, nil, nil, fmt.Errorf("the client and client-hardware-addr may not both be empty")
		}
		clientMAC, err := net.ParseMAC(clientHWAddr)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse MAC: %s", err)
		}
		client = generateEUI64Address(net.ParseIP("fe80::0"), clientMAC).String()
	}
	dst := net.ParseIP(client)
	if dst == nil {
		return nil, nil, nil, fmt.Errorf("the client IPv6 address (%s) is illegal", client)
	}
	if !dst.IsLinkLocalUnicast() {
		return nil, nil, nil, fmt.Errorf("the client IPv6 address should be a link-local address")
	}

	if prefix == "" {
		return nil, nil, nil, fmt.Errorf("the prefix may not be empty")
	}
	_, cidr, err := net.ParseCIDR(prefix)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("the prefix (%s) is illegal", prefix)
	}

	return src, dst, cidr, nil
}

func generateEUI64Address(prefix net.IP, mac net.HardwareAddr) net.IP {
	ip := make([]byte, 16)
	copy(ip[0:8], prefix[0:8])

	copy(ip[8:11], mac[0:3])
	ip[8] ^= 0x02
	ip[11] = 0xff
	ip[12] = 0xfe
	copy(ip[13:16], mac[3:6])

	return ip
}

func tryDiscoverNeighborMAC(ifaceName string, ip net.IP, retry int) (net.HardwareAddr, error) {
	for i := retry; i > 0; i-- {
		mac, err := discoverNeighborMAC(ifaceName, ip)
		if err != nil {
			return nil, err
		}
		if mac == nil {
			log.Println("INFO: retry in 5s")
			continue
		}
		return mac, nil
	}

	return nil, fmt.Errorf("failed to discover neighbor MAC. Try %d times", retry)
}

func discoverNeighborMAC(ifaceName string, ip net.IP) (net.HardwareAddr, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("get interface by name: %s", err)
	}
	conn, err := tryCreateNDPConn(iface, 5)
	if err != nil {
		return nil, fmt.Errorf("create NDP connection: %s", err)
	}
	defer conn.Close()

	target := netip.MustParseAddr(ip.String())
	solicitationAddr, err := ndp.SolicitedNodeMulticast(target)
	if err != nil {
		return nil, fmt.Errorf("determine solicited-node multicast address: %s", err)
	}
	solicitation := &ndp.NeighborSolicitation{
		TargetAddress: target,
		Options: []ndp.Option{
			&ndp.LinkLayerAddress{
				Direction: ndp.Source,
				Addr:      iface.HardwareAddr,
			},
		},
	}
	if err := conn.WriteTo(solicitation, nil, solicitationAddr); err != nil {
		return nil, fmt.Errorf("write neighbor solicitation: %s", err)
	}

	var f ipv6.ICMPFilter
	f.SetAll(true)
	f.Accept(ipv6.ICMPTypeNeighborAdvertisement)
	if err := conn.SetICMPFilter(&f); err != nil {
		return nil, fmt.Errorf("set ICMPv6 filter: %s", err)
	}

	msg, _, from, err := conn.ReadFrom()
	if err != nil {
		return nil, fmt.Errorf("read NDP message: %s", err)
	}
	if target.WithZone(ifaceName).Compare(from) != 0 && target.Compare(from) != 0 {
		log.Println("INFO: the NDP message is not from solicitation target")
		return nil, nil
	}
	advertisement := msg.(*ndp.NeighborAdvertisement)
	if len(advertisement.Options) != 1 {
		return nil, fmt.Errorf("get %d option(s) in neighbor advertisement, but expect one", len(advertisement.Options))
	}
	linkLayerAddr, ok := advertisement.Options[0].(*ndp.LinkLayerAddress)
	if !ok {
		return nil, fmt.Errorf("advertisement option is not a link-layer address")
	}
	return linkLayerAddr.Addr, nil
}

func tryCreateNDPConn(iface *net.Interface, retry int) (*ndp.Conn, error) {
	var err error
	for i := retry; i > 0; i-- {
		conn, _, err := ndp.Listen(iface, ndp.LinkLocal)
		if err != nil {
			// caused by tap device state down?
			log.Printf("Warnning: listen interface link-local address: %s. Retry in 5s\n", err)
			time.Sleep(5 * time.Second)
		}
		if err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("listen interface link-local address: %s. Retry %d times", err, retry)
}

func startRouteAdvertisement(ifaceName string, src net.IP, dst net.IP, cidr *net.IPNet) error {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("get interface by name: %s", err)
	}
	conn, err := tryCreateNDPConn(iface, 5)
	if err != nil {
		return fmt.Errorf("create NDP connection: %s", err)
	}
	defer conn.Close()

	var filter ipv6.ICMPFilter
	filter.SetAll(true)
	filter.Accept(ipv6.ICMPTypeRouterSolicitation)
	if err := conn.SetICMPFilter(&filter); err != nil {
		return fmt.Errorf("apply ICMPv6 filter: %s", err)
	}
	if err := conn.JoinGroup(linkLocalAllRouters); err != nil {
		return fmt.Errorf("join IPv6 link-local all routers multicast group: %s", err)
	}

	prefixLen, _ := cidr.Mask.Size()
	advertisement := &ndp.RouterAdvertisement{
		CurrentHopLimit:      255,
		RouterLifetime:       65535 * time.Second,
		ManagedConfiguration: true,
		OtherConfiguration:   true,
		Options: []ndp.Option{
			&ndp.PrefixInformation{
				PrefixLength:  uint8(prefixLen),
				Prefix:        netip.MustParseAddr(cidr.IP.String()),
				OnLink:        true,
				ValidLifetime: 4294967295 * time.Second,
			},
		},
	}

	controlMsg := &ipv6.ControlMessage{
		HopLimit: 255,
		Src:      src,
	}

	if src == nil {
		advertisement.RouterLifetime = 0
		controlMsg = nil
	}

	recivedRS := make(chan struct{}, 1)
	go func(recivedRS chan struct{}) {
		for {
			_, _, from, err := conn.ReadFrom()
			if err != nil {
				log.Printf("Warnning: read NDP message: %s. Retry in 5s\n", err)
				time.Sleep(5 * time.Second)
				continue
			}
			target := netip.MustParseAddr(dst.String())
			if target.WithZone(ifaceName).Compare(from) != 0 && target.Compare(from) != 0 {
				continue
			}
			recivedRS <- struct{}{}
		}
	}(recivedRS)

	raPeriod := time.NewTicker(time.Minute)
	cnt := 0
	for {
		select {
		case <-recivedRS:
			if err := conn.WriteTo(advertisement, controlMsg, netip.MustParseAddr(dst.String())); err != nil {
				return fmt.Errorf("send route advertisement: %s", err)
			}
			log.Printf("INFO: reply RS from %s\n", dst.String())
		case <-raPeriod.C:
			if err := conn.WriteTo(advertisement, controlMsg, netip.MustParseAddr(dst.String())); err != nil {
				return fmt.Errorf("send route advertisement: %s", err)
			}
			cnt++
			if cnt == 10 {
				log.Printf("INFO: send RA to %s 10 times\n", dst.String())
				cnt = 0
			}
		}
	}
}

func executeCommand(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%q: %s: %s", cmd.String(), err, output)
	}
	return string(output), nil
}

func init() {
	flag.StringVar(&iface, "interface", "", "The interface to listen to.")
	flag.StringVar(&router, "router", "", "The IPv6 address of the default router. "+
		"It's recommanded to use the link-local address of the router, "+
		"otherwise the SLAAC link-local address formed by router hardware address will be used.")
	flag.BoolVar(&isRemoteRoute, "is-remote-route", false, "")
	flag.StringVar(&client, "client", "", "The IPv6 link-local address of the client to advertise to. "+
		"The SLAAC link-local address formed by client hardware address will be used when empty.")
	flag.StringVar(&clientHWAddr, "client-hardware-addr", "", "The hardware address of the client.")
	flag.StringVar(&prefix, "prefix", "", "The prefix of the subnet.")

	flag.Parse()
}

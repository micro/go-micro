package addr

import (
	"fmt"
	"net"
)

var (
	privateBlocks []*net.IPNet
)

func init() {
	blocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10",
		"fd00::/8",
	}
	AppendPrivateBlocks(blocks...)
}

// AppendPrivateBlocks append private network blocks
func AppendPrivateBlocks(bs ...string) {
	for _, b := range bs {
		if _, block, err := net.ParseCIDR(b); err == nil {
			privateBlocks = append(privateBlocks, block)
		}
	}
}

func isPrivateIP(ipAddr string) bool {
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return false
	}

	for _, blocks := range privateBlocks {
		if blocks.Contains(ip) {
			return true
		}
	}
	return false
}

func addrToIP(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPAddr:
		return v.IP
	case *net.IPNet:
		return v.IP
	default:
		return nil
	}
}

func localIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var ipAddrs []string

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue // ignore error
		}

		for _, addr := range addrs {
			if ip := addrToIP(addr); ip != nil {
				ipAddrs = append(ipAddrs, ip.String())
			}
		}
	}

	return ipAddrs
}

// IsLocal tells us whether an ip is local
func IsLocal(addr string) bool {
	// extract the host
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		addr = host
	}

	// check if its localhost
	if addr == "localhost" {
		return true
	}

	// check against all local ips
	for _, ip := range localIPs() {
		if addr == ip {
			return true
		}
	}

	return false
}

// Extract returns a real ip
func Extract(addr string) (string, error) {
	// if addr specified then its returned
	if len(addr) > 0 && (addr != "0.0.0.0" && addr != "[::]" && addr != "::") {
		return addr, nil
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("Failed to get interfaces! Err: %v", err)
	}

	//nolint:prealloc
	var addrs []net.Addr
	var loAddrs []net.Addr
	for _, iface := range ifaces {
		ifaceAddrs, err := iface.Addrs()
		if err != nil {
			// ignore error, interface can disappear from system
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			loAddrs = append(loAddrs, ifaceAddrs...)
			continue
		}
		addrs = append(addrs, ifaceAddrs...)
	}
	addrs = append(addrs, loAddrs...)

	var ipAddr string
	var publicIP string

	for _, rawAddr := range addrs {
		ip := addrToIP(rawAddr)
		if ip == nil {
			continue
		}

		if !isPrivateIP(ip.String()) {
			publicIP = ip.String()
			continue
		}

		ipAddr = ip.String()
		break
	}

	// return private ip
	if len(ipAddr) > 0 {
		a := net.ParseIP(ipAddr)
		if a == nil {
			return "", fmt.Errorf("ip addr %s is invalid", ipAddr)
		}
		return a.String(), nil
	}

	// return public or virtual ip
	if len(publicIP) > 0 {
		a := net.ParseIP(publicIP)
		if a == nil {
			return "", fmt.Errorf("ip addr %s is invalid", publicIP)
		}
		return a.String(), nil
	}

	return "", fmt.Errorf("No IP address found, and explicit IP not provided")
}

// IPs returns all known ips
func IPs() []string {
	return localIPs()
}

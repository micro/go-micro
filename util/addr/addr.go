// addr provides functions to retrieve local IP addresses from device interfaces.
package addr

import (
	"net"

	"github.com/pkg/errors"
)

var (
	// ErrIPNotFound no IP address found, and explicit IP not provided.
	ErrIPNotFound = errors.New("no IP address found, and explicit IP not provided")
)

// IsLocal checks whether an IP belongs to one of the device's interfaces.
func IsLocal(addr string) bool {
	// Extract the host
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		addr = host
	}

	if addr == "localhost" {
		return true
	}

	// Check against all local ips
	for _, ip := range IPs() {
		if addr == ip {
			return true
		}
	}

	return false
}

// Extract returns a valid IP address. If the address provided is a valid
// address, it will be returned directly. Otherwise the available interfaces
// be itterated over to find an IP address, prefferably private.
func Extract(addr string) (string, error) {
	// if addr is already specified then it's directly returned
	if len(addr) > 0 && (addr != "0.0.0.0" && addr != "[::]" && addr != "::") {
		return addr, nil
	}

	var (
		addrs   []net.Addr
		loAddrs []net.Addr
	)

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Wrap(err, "failed to get interfaces")
	}

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

	// Add loopback addresses to the end of the list
	addrs = append(addrs, loAddrs...)

	// Try to find private IP in list, public IP otherwise
	ip, err := findIP(addrs)
	if err != nil {
		return "", err
	}

	return ip.String(), nil
}

// IPs returns all available interface IP addresses.
func IPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var ipAddrs []string

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil {
				continue
			}

			ipAddrs = append(ipAddrs, ip.String())
		}
	}

	return ipAddrs
}

// findIP will return the first private IP available in the list,
// if no private IP is available it will return a public IP if present.
func findIP(addresses []net.Addr) (net.IP, error) {
	var publicIP net.IP

	for _, rawAddr := range addresses {
		var ip net.IP
		switch addr := rawAddr.(type) {
		case *net.IPAddr:
			ip = addr.IP
		case *net.IPNet:
			ip = addr.IP
		default:
			continue
		}

		if !ip.IsPrivate() {
			publicIP = ip
			continue
		}

		// Return private IP if available
		return ip, nil
	}

	// Return public or virtual IP
	if len(publicIP) > 0 {
		return publicIP, nil
	}

	return nil, ErrIPNotFound
}

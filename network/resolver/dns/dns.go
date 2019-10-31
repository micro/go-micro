// Package dns resolves names to dns records
package dns

import (
	"net"

	"github.com/micro/go-micro/network/resolver"
)

// Resolver is a DNS network resolve
type Resolver struct{}

// Resolve assumes ID is a domain name e.g micro.mu
func (r *Resolver) Resolve(name string) ([]*resolver.Record, error) {
	host, port, err := net.SplitHostPort(name)
	if err != nil {
		host = name
		port = "8085"
	}

	if len(host) == 0 {
		host = "localhost"
	}

	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil, err
	}

	records := make([]*resolver.Record, 0, len(addrs))

	for _, addr := range addrs {
		// join resolved record with port
		address := net.JoinHostPort(addr, port)
		// append to record set
		records = append(records, &resolver.Record{
			Address: address,
		})
	}

	return records, nil
}

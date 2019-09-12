// Package dns resolves names to dns srv records
package dns

import (
	"fmt"
	"net"

	"github.com/micro/go-micro/network/resolver"
)

// Resolver is a DNS network resolve
type Resolver struct{}

// Resolve assumes ID is a domain name e.g micro.mu
func (r *Resolver) Resolve(name string) ([]*resolver.Record, error) {
	_, addrs, err := net.LookupSRV("network", "udp", name)
	if err != nil {
		return nil, err
	}

	records := make([]*resolver.Record, len(addrs))

	for i, addr := range addrs {
		address := addr.Target
		if addr.Port > 0 {
			address = fmt.Sprintf("%s:%d", addr.Target, addr.Port)
		}
		records[i] = &resolver.Record{Address: address}
	}

	return records, nil
}

// Package dns resolves names to dns records
package dns

import (
	"context"
	"net"

	"github.com/micro/go-micro/v2/network/resolver"
	"github.com/miekg/dns"
)

// Resolver is a DNS network resolve
type Resolver struct {
	// The resolver address to use
	Address string
}

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

	if len(r.Address) == 0 {
		r.Address = "1.0.0.1:53"
	}

	//nolint:prealloc
	var records []*resolver.Record

	// parsed an actual ip
	if v := net.ParseIP(host); v != nil {
		records = append(records, &resolver.Record{
			Address: net.JoinHostPort(host, port),
		})
		return records, nil
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeA)
	rec, err := dns.ExchangeContext(context.Background(), m, r.Address)
	if err != nil {
		return nil, err
	}

	for _, answer := range rec.Answer {
		h := answer.Header()
		// check record type matches
		if h.Rrtype != dns.TypeA {
			continue
		}

		arec, _ := answer.(*dns.A)
		addr := arec.A.String()

		// join resolved record with port
		address := net.JoinHostPort(addr, port)
		// append to record set
		records = append(records, &resolver.Record{
			Address: address,
		})
	}

	// no records returned so just best effort it
	if len(records) == 0 {
		records = append(records, &resolver.Record{
			Address: net.JoinHostPort(host, port),
		})
	}

	return records, nil
}

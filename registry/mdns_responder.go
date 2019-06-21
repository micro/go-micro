package registry

import (
	"strings"

	"github.com/micro/go-micro/util/log"
	"github.com/miekg/dns"
)

func (m *mdnsRegistry) Records(q dns.Question) []dns.RR {
	var err error
	var ret []dns.RR
	var rr []dns.RR

	m.RLock()
	svcs, ok := m.services[strings.TrimSuffix(q.Name, "."+m.domain+".")]
	if !ok {
		m.RUnlock()
		return nil
	}
	m.RUnlock()

	for _, s := range svcs {
		switch q.Qtype {
		case dns.TypeTXT:
			ret, err = m.rrTXT(s, 60)
		case dns.TypePTR, dns.TypeSRV:
			rrptr, err := m.rrPTR(s, 60)
			if err != nil {
				break
			}
			rrtxt, err := m.rrTXT(s, 60)
			if err != nil {
				break
			}
			rrsrv, err := m.rrSRV(s, 60)
			if err != nil {
				break
			}
			rra, err := m.rrA(s, 60)
			if err != nil {
				break
			}
			ret = append(rrptr, rrtxt...)
			ret = append(ret, rrsrv...)
			ret = append(ret, rra...)
		}

		if err != nil {
			log.Logf("[mdns] Failed to respond: %v", err)
		}

		rr = append(rr, ret...)
	}

	return rr
}

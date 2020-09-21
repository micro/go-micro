package mdns

import (
	"reflect"
	"testing"
)
import "github.com/miekg/dns"

type mockMDNSService struct{}

func (s *mockMDNSService) Records(q dns.Question) []dns.RR {
	return []dns.RR{
		&dns.PTR{
			Hdr: dns.RR_Header{
				Name:   "fakerecord",
				Rrtype: dns.TypePTR,
				Class:  dns.ClassINET,
				Ttl:    42,
			},
			Ptr: "fake.local.",
		},
	}
}

func (s *mockMDNSService) Announcement() []dns.RR {
	return []dns.RR{
		&dns.PTR{
			Hdr: dns.RR_Header{
				Name:   "fakeannounce",
				Rrtype: dns.TypePTR,
				Class:  dns.ClassINET,
				Ttl:    42,
			},
			Ptr: "fake.local.",
		},
	}
}

func TestDNSSDServiceRecords(t *testing.T) {
	s := &DNSSDService{
		MDNSService: &MDNSService{
			serviceAddr: "_foobar._tcp.local.",
			Domain:      "local",
		},
	}
	q := dns.Question{
		Name:   "_services._dns-sd._udp.local.",
		Qtype:  dns.TypePTR,
		Qclass: dns.ClassINET,
	}
	recs := s.Records(q)
	if got, want := len(recs), 1; got != want {
		t.Fatalf("s.Records(%v) returned %v records, want %v", q, got, want)
	}

	want := dns.RR(&dns.PTR{
		Hdr: dns.RR_Header{
			Name:   "_services._dns-sd._udp.local.",
			Rrtype: dns.TypePTR,
			Class:  dns.ClassINET,
			Ttl:    defaultTTL,
		},
		Ptr: "_foobar._tcp.local.",
	})
	if got := recs[0]; !reflect.DeepEqual(got, want) {
		t.Errorf("s.Records()[0] = %v, want %v", got, want)
	}
}

package mdns

import "github.com/miekg/dns"

// DNSSDService is a service that complies with the DNS-SD (RFC 6762) and MDNS
// (RFC 6762) specs for local, multicast-DNS-based discovery.
//
// DNSSDService implements the Zone interface and wraps an MDNSService instance.
// To deploy an mDNS service that is compliant with DNS-SD, it's recommended to
// register only the wrapped instance with the server.
//
// Example usage:
//     service := &mdns.DNSSDService{
//       MDNSService: &mdns.MDNSService{
// 	       Instance: "My Foobar Service",
// 	       Service: "_foobar._tcp",
// 	       Port:    8000,
//        }
//      }
//      server, err := mdns.NewServer(&mdns.Config{Zone: service})
//      if err != nil {
//        log.Fatalf("Error creating server: %v", err)
//      }
//      defer server.Shutdown()
type DNSSDService struct {
	MDNSService *MDNSService
}

// Records returns DNS records in response to a DNS question.
//
// This function returns the DNS response of the underlying MDNSService
// instance.  It also returns a PTR record for a request for "
// _services._dns-sd._udp.<Domain>", as described in section 9 of RFC 6763
// ("Service Type Enumeration"), to allow browsing of the underlying MDNSService
// instance.
func (s *DNSSDService) Records(q dns.Question) []dns.RR {
	var recs []dns.RR
	if q.Name == "_services._dns-sd._udp."+s.MDNSService.Domain+"." {
		recs = s.dnssdMetaQueryRecords(q)
	}
	return append(recs, s.MDNSService.Records(q)...)
}

// dnssdMetaQueryRecords returns the DNS records in response to a "meta-query"
// issued to browse for DNS-SD services, as per section 9. of RFC6763.
//
// A meta-query has a name of the form "_services._dns-sd._udp.<Domain>" where
// Domain is a fully-qualified domain, such as "local."
func (s *DNSSDService) dnssdMetaQueryRecords(q dns.Question) []dns.RR {
	// Intended behavior, as described in the RFC:
	//     ...it may be useful for network administrators to find the list of
	//     advertised service types on the network, even if those Service Names
	//     are just opaque identifiers and not particularly informative in
	//     isolation.
	//
	//     For this purpose, a special meta-query is defined.  A DNS query for PTR
	//     records with the name "_services._dns-sd._udp.<Domain>" yields a set of
	//     PTR records, where the rdata of each PTR record is the two-abel
	//     <Service> name, plus the same domain, e.g., "_http._tcp.<Domain>".
	//     Including the domain in the PTR rdata allows for slightly better name
	//     compression in Unicast DNS responses, but only the first two labels are
	//     relevant for the purposes of service type enumeration.  These two-label
	//     service types can then be used to construct subsequent Service Instance
	//     Enumeration PTR queries, in this <Domain> or others, to discover
	//     instances of that service type.
	return []dns.RR{
		&dns.PTR{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypePTR,
				Class:  dns.ClassINET,
				Ttl:    defaultTTL,
			},
			Ptr: s.MDNSService.serviceAddr,
		},
	}
}

// Announcement returns DNS records that should be broadcast during the initial
// availability of the service, as described in section 8.3 of RFC 6762.
// TODO(reddaly): Add this when Announcement is added to the mdns.Zone interface.
//func (s *DNSSDService) Announcement() []dns.RR {
//	return s.MDNSService.Announcement()
//}

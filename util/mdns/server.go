package mdns

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/micro/go-micro/v2/logger"
	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

var (
	mdnsGroupIPv4 = net.ParseIP("224.0.0.251")
	mdnsGroupIPv6 = net.ParseIP("ff02::fb")

	// mDNS wildcard addresses
	mdnsWildcardAddrIPv4 = &net.UDPAddr{
		IP:   net.ParseIP("224.0.0.0"),
		Port: 5353,
	}
	mdnsWildcardAddrIPv6 = &net.UDPAddr{
		IP:   net.ParseIP("ff02::"),
		Port: 5353,
	}

	// mDNS endpoint addresses
	ipv4Addr = &net.UDPAddr{
		IP:   mdnsGroupIPv4,
		Port: 5353,
	}
	ipv6Addr = &net.UDPAddr{
		IP:   mdnsGroupIPv6,
		Port: 5353,
	}
)

// GetMachineIP is a func which returns the outbound IP of this machine.
// Used by the server to determine whether to attempt send the response on a local address
type GetMachineIP func() net.IP

// Config is used to configure the mDNS server
type Config struct {
	// Zone must be provided to support responding to queries
	Zone Zone

	// Iface if provided binds the multicast listener to the given
	// interface. If not provided, the system default multicase interface
	// is used.
	Iface *net.Interface

	// Port If it is not 0, replace the port 5353 with this port number.
	Port int

	// GetMachineIP is a function to return the IP of the local machine
	GetMachineIP GetMachineIP
	// LocalhostChecking if enabled asks the server to also send responses to 0.0.0.0 if the target IP
	// is this host (as defined by GetMachineIP). Useful in case machine is on a VPN which blocks comms on non standard ports
	LocalhostChecking bool
}

// Server is an mDNS server used to listen for mDNS queries and respond if we
// have a matching local record
type Server struct {
	config *Config

	ipv4List *net.UDPConn
	ipv6List *net.UDPConn

	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex
	wg           sync.WaitGroup

	outboundIP net.IP
}

// NewServer is used to create a new mDNS server from a config
func NewServer(config *Config) (*Server, error) {
	setCustomPort(config.Port)

	// Create the listeners
	// Create wildcard connections (because :5353 can be already taken by other apps)
	ipv4List, _ := net.ListenUDP("udp4", mdnsWildcardAddrIPv4)
	ipv6List, _ := net.ListenUDP("udp6", mdnsWildcardAddrIPv6)
	if ipv4List == nil && ipv6List == nil {
		return nil, fmt.Errorf("[ERR] mdns: Failed to bind to any udp port!")
	}

	if ipv4List == nil {
		ipv4List = &net.UDPConn{}
	}
	if ipv6List == nil {
		ipv6List = &net.UDPConn{}
	}

	// Join multicast groups to receive announcements
	p1 := ipv4.NewPacketConn(ipv4List)
	p2 := ipv6.NewPacketConn(ipv6List)
	p1.SetMulticastLoopback(true)
	p2.SetMulticastLoopback(true)

	if config.Iface != nil {
		if err := p1.JoinGroup(config.Iface, &net.UDPAddr{IP: mdnsGroupIPv4}); err != nil {
			return nil, err
		}
		if err := p2.JoinGroup(config.Iface, &net.UDPAddr{IP: mdnsGroupIPv6}); err != nil {
			return nil, err
		}
	} else {
		ifaces, err := net.Interfaces()
		if err != nil {
			return nil, err
		}
		errCount1, errCount2 := 0, 0
		for _, iface := range ifaces {
			if err := p1.JoinGroup(&iface, &net.UDPAddr{IP: mdnsGroupIPv4}); err != nil {
				errCount1++
			}
			if err := p2.JoinGroup(&iface, &net.UDPAddr{IP: mdnsGroupIPv6}); err != nil {
				errCount2++
			}
		}
		if len(ifaces) == errCount1 && len(ifaces) == errCount2 {
			return nil, fmt.Errorf("Failed to join multicast group on all interfaces!")
		}
	}

	ipFunc := getOutboundIP
	if config.GetMachineIP != nil {
		ipFunc = config.GetMachineIP
	}

	s := &Server{
		config:     config,
		ipv4List:   ipv4List,
		ipv6List:   ipv6List,
		shutdownCh: make(chan struct{}),
		outboundIP: ipFunc(),
	}

	go s.recv(s.ipv4List)
	go s.recv(s.ipv6List)

	s.wg.Add(1)
	go s.probe()

	return s, nil
}

// Shutdown is used to shutdown the listener
func (s *Server) Shutdown() error {
	s.shutdownLock.Lock()
	defer s.shutdownLock.Unlock()

	if s.shutdown {
		return nil
	}

	s.shutdown = true
	close(s.shutdownCh)
	s.unregister()

	if s.ipv4List != nil {
		s.ipv4List.Close()
	}
	if s.ipv6List != nil {
		s.ipv6List.Close()
	}

	s.wg.Wait()
	return nil
}

// recv is a long running routine to receive packets from an interface
func (s *Server) recv(c *net.UDPConn) {
	if c == nil {
		return
	}
	buf := make([]byte, 65536)
	for {
		s.shutdownLock.Lock()
		if s.shutdown {
			s.shutdownLock.Unlock()
			return
		}
		s.shutdownLock.Unlock()
		n, from, err := c.ReadFrom(buf)
		if err != nil {
			continue
		}
		if err := s.parsePacket(buf[:n], from); err != nil {
			log.Errorf("[ERR] mdns: Failed to handle query: %v", err)
		}
	}
}

// parsePacket is used to parse an incoming packet
func (s *Server) parsePacket(packet []byte, from net.Addr) error {
	var msg dns.Msg
	if err := msg.Unpack(packet); err != nil {
		log.Errorf("[ERR] mdns: Failed to unpack packet: %v", err)
		return err
	}
	// TODO: This is a bit of a hack
	// We decided to ignore some mDNS answers for the time being
	// See: https://tools.ietf.org/html/rfc6762#section-7.2
	msg.Truncated = false
	return s.handleQuery(&msg, from)
}

// handleQuery is used to handle an incoming query
func (s *Server) handleQuery(query *dns.Msg, from net.Addr) error {
	if query.Opcode != dns.OpcodeQuery {
		// "In both multicast query and multicast response messages, the OPCODE MUST
		// be zero on transmission (only standard queries are currently supported
		// over multicast).  Multicast DNS messages received with an OPCODE other
		// than zero MUST be silently ignored."  Note: OpcodeQuery == 0
		return fmt.Errorf("mdns: received query with non-zero Opcode %v: %v", query.Opcode, *query)
	}
	if query.Rcode != 0 {
		// "In both multicast query and multicast response messages, the Response
		// Code MUST be zero on transmission.  Multicast DNS messages received with
		// non-zero Response Codes MUST be silently ignored."
		return fmt.Errorf("mdns: received query with non-zero Rcode %v: %v", query.Rcode, *query)
	}

	// TODO(reddaly): Handle "TC (Truncated) Bit":
	//    In query messages, if the TC bit is set, it means that additional
	//    Known-Answer records may be following shortly.  A responder SHOULD
	//    record this fact, and wait for those additional Known-Answer records,
	//    before deciding whether to respond.  If the TC bit is clear, it means
	//    that the querying host has no additional Known Answers.
	if query.Truncated {
		return fmt.Errorf("[ERR] mdns: support for DNS requests with high truncated bit not implemented: %v", *query)
	}

	var unicastAnswer, multicastAnswer []dns.RR

	// Handle each question
	for _, q := range query.Question {
		mrecs, urecs := s.handleQuestion(q)
		multicastAnswer = append(multicastAnswer, mrecs...)
		unicastAnswer = append(unicastAnswer, urecs...)
	}

	// See section 18 of RFC 6762 for rules about DNS headers.
	resp := func(unicast bool) *dns.Msg {
		// 18.1: ID (Query Identifier)
		// 0 for multicast response, query.Id for unicast response
		id := uint16(0)
		if unicast {
			id = query.Id
		}

		var answer []dns.RR
		if unicast {
			answer = unicastAnswer
		} else {
			answer = multicastAnswer
		}
		if len(answer) == 0 {
			return nil
		}

		return &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Id: id,

				// 18.2: QR (Query/Response) Bit - must be set to 1 in response.
				Response: true,

				// 18.3: OPCODE - must be zero in response (OpcodeQuery == 0)
				Opcode: dns.OpcodeQuery,

				// 18.4: AA (Authoritative Answer) Bit - must be set to 1
				Authoritative: true,

				// The following fields must all be set to 0:
				// 18.5: TC (TRUNCATED) Bit
				// 18.6: RD (Recursion Desired) Bit
				// 18.7: RA (Recursion Available) Bit
				// 18.8: Z (Zero) Bit
				// 18.9: AD (Authentic Data) Bit
				// 18.10: CD (Checking Disabled) Bit
				// 18.11: RCODE (Response Code)
			},
			// 18.12 pertains to questions (handled by handleQuestion)
			// 18.13 pertains to resource records (handled by handleQuestion)

			// 18.14: Name Compression - responses should be compressed (though see
			// caveats in the RFC), so set the Compress bit (part of the dns library
			// API, not part of the DNS packet) to true.
			Compress: true,
			Question: query.Question,
			Answer:   answer,
		}
	}

	if mresp := resp(false); mresp != nil {
		if err := s.sendResponse(mresp, from); err != nil {
			return fmt.Errorf("mdns: error sending multicast response: %v", err)
		}
	}
	if uresp := resp(true); uresp != nil {
		if err := s.sendResponse(uresp, from); err != nil {
			return fmt.Errorf("mdns: error sending unicast response: %v", err)
		}
	}
	return nil
}

// handleQuestion is used to handle an incoming question
//
// The response to a question may be transmitted over multicast, unicast, or
// both.  The return values are DNS records for each transmission type.
func (s *Server) handleQuestion(q dns.Question) (multicastRecs, unicastRecs []dns.RR) {
	records := s.config.Zone.Records(q)
	if len(records) == 0 {
		return nil, nil
	}

	// Handle unicast and multicast responses.
	// TODO(reddaly): The decision about sending over unicast vs. multicast is not
	// yet fully compliant with RFC 6762.  For example, the unicast bit should be
	// ignored if the records in question are close to TTL expiration.  For now,
	// we just use the unicast bit to make the decision, as per the spec:
	//     RFC 6762, section 18.12.  Repurposing of Top Bit of qclass in Question
	//     Section
	//
	//     In the Question Section of a Multicast DNS query, the top bit of the
	//     qclass field is used to indicate that unicast responses are preferred
	//     for this particular question.  (See Section 5.4.)
	if q.Qclass&(1<<15) != 0 {
		return nil, records
	}
	return records, nil
}

func (s *Server) probe() {
	defer s.wg.Done()

	sd, ok := s.config.Zone.(*MDNSService)
	if !ok {
		return
	}

	name := fmt.Sprintf("%s.%s.%s.", sd.Instance, trimDot(sd.Service), trimDot(sd.Domain))

	q := new(dns.Msg)
	q.SetQuestion(name, dns.TypePTR)
	q.RecursionDesired = false

	srv := &dns.SRV{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeSRV,
			Class:  dns.ClassINET,
			Ttl:    defaultTTL,
		},
		Priority: 0,
		Weight:   0,
		Port:     uint16(sd.Port),
		Target:   sd.HostName,
	}
	txt := &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    defaultTTL,
		},
		Txt: sd.TXT,
	}
	q.Ns = []dns.RR{srv, txt}

	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 3; i++ {
		if err := s.SendMulticast(q); err != nil {
			log.Errorf("[ERR] mdns: failed to send probe:", err.Error())
		}
		time.Sleep(time.Duration(randomizer.Intn(250)) * time.Millisecond)
	}

	resp := new(dns.Msg)
	resp.MsgHdr.Response = true

	// set for query
	q.SetQuestion(name, dns.TypeANY)

	resp.Answer = append(resp.Answer, s.config.Zone.Records(q.Question[0])...)

	// reset
	q.SetQuestion(name, dns.TypePTR)

	// From RFC6762
	//    The Multicast DNS responder MUST send at least two unsolicited
	//    responses, one second apart. To provide increased robustness against
	//    packet loss, a responder MAY send up to eight unsolicited responses,
	//    provided that the interval between unsolicited responses increases by
	//    at least a factor of two with every response sent.
	timeout := 1 * time.Second
	timer := time.NewTimer(timeout)
	for i := 0; i < 3; i++ {
		if err := s.SendMulticast(resp); err != nil {
			log.Errorf("[ERR] mdns: failed to send announcement:", err.Error())
		}
		select {
		case <-timer.C:
			timeout *= 2
			timer.Reset(timeout)
		case <-s.shutdownCh:
			timer.Stop()
			return
		}
	}
}

// SendMulticast us used to send a multicast response packet
func (s *Server) SendMulticast(msg *dns.Msg) error {
	buf, err := msg.Pack()
	if err != nil {
		return err
	}
	if s.ipv4List != nil {
		s.ipv4List.WriteToUDP(buf, ipv4Addr)
	}
	if s.ipv6List != nil {
		s.ipv6List.WriteToUDP(buf, ipv6Addr)
	}
	return nil
}

// sendResponse is used to send a response packet
func (s *Server) sendResponse(resp *dns.Msg, from net.Addr) error {
	// TODO(reddaly): Respect the unicast argument, and allow sending responses
	// over multicast.
	buf, err := resp.Pack()
	if err != nil {
		return err
	}

	// Determine the socket to send from
	addr := from.(*net.UDPAddr)
	conn := s.ipv4List
	backupTarget := net.IPv4zero

	if addr.IP.To4() == nil {
		conn = s.ipv6List
		backupTarget = net.IPv6zero
	}
	_, err = conn.WriteToUDP(buf, addr)
	// If the address we're responding to is this machine then we can also attempt sending on 0.0.0.0
	// This covers the case where this machine is using a VPN and certain ports are blocked so the response never gets there
	// Sending two responses is OK
	if s.config.LocalhostChecking && addr.IP.Equal(s.outboundIP) {
		// ignore any errors, this is best efforts
		conn.WriteToUDP(buf, &net.UDPAddr{IP: backupTarget, Port: addr.Port})
	}
	return err

}

func (s *Server) unregister() error {
	sd, ok := s.config.Zone.(*MDNSService)
	if !ok {
		return nil
	}

	atomic.StoreUint32(&sd.TTL, 0)
	name := fmt.Sprintf("%s.%s.%s.", sd.Instance, trimDot(sd.Service), trimDot(sd.Domain))

	q := new(dns.Msg)
	q.SetQuestion(name, dns.TypeANY)

	resp := new(dns.Msg)
	resp.MsgHdr.Response = true
	resp.Answer = append(resp.Answer, s.config.Zone.Records(q.Question[0])...)

	return s.SendMulticast(resp)
}

func setCustomPort(port int) {
	if port != 0 {
		if mdnsWildcardAddrIPv4.Port != port {
			mdnsWildcardAddrIPv4.Port = port
		}
		if mdnsWildcardAddrIPv6.Port != port {
			mdnsWildcardAddrIPv6.Port = port
		}
		if ipv4Addr.Port != port {
			ipv4Addr.Port = port
		}
		if ipv6Addr.Port != port {
			ipv6Addr.Port = port
		}
	}
}

// getOutboundIP returns the IP address of this machine as seen when dialling out
func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// no net connectivity maybe so fallback
		return nil
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

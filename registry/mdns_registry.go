// Package mdns is a multicast dns registry
package registry

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/util/log"
	"github.com/micro/mdns"
	"github.com/miekg/dns"
)

type mdnsTxt struct {
	Version   string
	Endpoints []*Endpoint
	Metadata  map[string]string
}

type mdnsRegistry struct {
	opts   Options
	domain string
	iface  *net.Interface
	sync.RWMutex
	services map[string][]*Service
	nodes    map[string]*Node
	srv      *mdns.Server
	updates  chan *dns.Msg
	err      error
}

func newRegistry(opts ...Option) Registry {
	options := Options{
		Timeout: 1 * time.Second,
	}

	for _, o := range opts {
		o(&options)
	}

	if options.Context == nil {
		options.Context = context.Background()
	}

	m := &mdnsRegistry{
		opts:     options,
		services: make(map[string][]*Service),
		updates:  make(chan *dns.Msg, 100),
		domain:   "local",
	}

	if domain, ok := options.Context.Value(mdnsDomainKey{}).(string); ok {
		m.domain = domain
	}

	cfg := &mdns.Config{
		Zone: m,
	}

	srv, err := mdns.NewServer(cfg)
	if err != nil {
		log.Fatalf("[mdns] Failed to initialize registry: %v", err)
	}

	m.Lock()
	m.srv = srv
	m.Unlock()

	//go m.run()

	return m
}

func (m *mdnsRegistry) Init(opts ...Option) error {
	m.Lock()
	defer m.Unlock()
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}

func (m *mdnsRegistry) Options() Options {
	m.RLock()
	defer m.RUnlock()
	return m.opts
}

func (m *mdnsRegistry) rrTXT(s *Service, ttl uint32) ([]dns.RR, error) {
	var rr []dns.RR

	m.RLock()
	domain := m.domain
	m.RUnlock()

	enctxt, err := encode(&mdnsTxt{
		Version:   s.Version,
		Endpoints: s.Endpoints,
		Metadata:  s.Metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("[mdns] Failed to register: %v", err)
	}

	for _, n := range s.Nodes {
		txt := &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   fmt.Sprintf("%s.%s.", n.Id, domain),
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Txt: enctxt,
		}
		rr = append(rr, txt)
	}

	return rr, nil
}

func (m *mdnsRegistry) rrSRV(s *Service, ttl uint32) ([]dns.RR, error) {
	var rr []dns.RR

	m.RLock()
	domain := m.domain
	m.RUnlock()

	for _, n := range s.Nodes {
		srv := &dns.SRV{
			Hdr: dns.RR_Header{
				Name:   fmt.Sprintf("%s.%s.", s.Name, domain),
				Rrtype: dns.TypeSRV,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Priority: 0,
			Weight:   0,
			Port:     uint16(n.Port),
			Target:   fmt.Sprintf("%s.%s.", n.Id, domain),
		}
		rr = append(rr, srv)
	}

	return rr, nil
}

func (m *mdnsRegistry) rrA(s *Service, ttl uint32) ([]dns.RR, error) {
	var rr []dns.RR

	m.RLock()
	domain := m.domain
	m.RUnlock()

	for _, n := range s.Nodes {
		a := &dns.A{
			Hdr: dns.RR_Header{
				Name:   fmt.Sprintf("%s.%s.", n.Id, domain),
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: net.ParseIP(n.Address),
		}
		rr = append(rr, a)
	}
	return rr, nil
}

func (m *mdnsRegistry) rrPTR(s *Service, ttl uint32) ([]dns.RR, error) {
	var rr []dns.RR

	m.RLock()
	domain := m.domain
	m.RUnlock()

	for _, n := range s.Nodes {
		ptr := &dns.PTR{
			Hdr: dns.RR_Header{
				Name:   fmt.Sprintf("%s.%s.", s.Name, domain),
				Rrtype: dns.TypePTR,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Ptr: fmt.Sprintf("%s.%s.", n.Id, domain),
		}
		rr = append(rr, ptr)
	}

	return rr, nil
}

func (m *mdnsRegistry) Register(s *Service, opts ...RegisterOption) error {
	m.Lock()
	if service, ok := m.services[s.Name]; !ok {
		m.services[s.Name] = []*Service{s}
	} else {
		m.services[s.Name] = addServices(service, []*Service{s})
	}
	m.Unlock()

	var options RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	ttl := uint32(options.TTL / time.Second)

	if options.TTL < 60 {
		ttl = 60
	}

	pkt := new(dns.Msg)
	pkt.MsgHdr.Response = true

	var rr []dns.RR
	var err error

	if rr, err = m.rrSRV(s, ttl); err != nil {
		return fmt.Errorf("[mdns] Failed to register: %v", err)
	} else {
		pkt.Answer = append(pkt.Answer, rr...)
	}

	if rr, err = m.rrA(s, ttl); err != nil {
		return fmt.Errorf("[mdns] Failed to register: %v", err)
	} else {
		pkt.Answer = append(pkt.Answer, rr...)
	}

	if rr, err = m.rrTXT(s, ttl); err != nil {
		return fmt.Errorf("[mdns] Failed to register: %v", err)
	} else {
		pkt.Answer = append(pkt.Answer, rr...)
	}

	_ = pkt
	//	m.updates <- pkt

	return nil
}

func (m *mdnsRegistry) run() {

	// not announce faster when 10 times at minute
	// https://tools.ietf.org/html/rfc6762#section-8.4
	ticker := time.NewTicker(time.Minute / 10)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pkt := <-m.updates
			if err := m.srv.SendMulticast(pkt); err != nil {
				log.Logf("[mdns] Failed to announce: %v", err)
			}
		}
	}
}

func (m *mdnsRegistry) Deregister(s *Service) error {
	m.Lock()
	if service, ok := m.services[s.Name]; ok {
		if services := delServices(service, []*Service{s}); len(services) == 0 {
			delete(m.services, s.Name)
		} else {
			m.services[s.Name] = services
		}
	}
	m.Unlock()

	return nil
}

func (m *mdnsRegistry) GetService(service string) ([]*Service, error) {
	serviceMap := make(map[string]*Service)
	entries := make(chan *mdns.ServiceEntry, 100)
	done := make(chan bool)

	p := mdns.DefaultParams(service)
	m.RLock()
	if len(m.domain) > 0 {
		p.Domain = m.domain
	}
	if m.iface != nil {
		p.Interface = m.iface
	}
	timeout := m.opts.Timeout
	m.RUnlock()

	// set context with timeout
	p.Context, _ = context.WithTimeout(context.Background(), timeout)
	// set entries channel
	p.Entries = entries

	go func() {
		for {
			select {
			case e := <-entries:
				// list record so skip
				if p.Service == "_services" {
					continue
				}

				if e.TTL == 0 {
					continue
				}
				txt, err := decode(e.InfoFields)
				if err != nil {
					continue
				}

				if strings.TrimSuffix(e.Name, "."+p.Domain+".") != service {
					continue
				}

				s, ok := serviceMap[txt.Version]
				if !ok {
					s = &Service{
						Name:      strings.TrimSuffix(e.Name, "."+p.Domain+"."),
						Version:   txt.Version,
						Endpoints: txt.Endpoints,
					}
				}

				s.Nodes = append(s.Nodes, &Node{
					Id:       strings.TrimSuffix(e.Host, "."+p.Domain+"."),
					Address:  e.AddrV4.String(),
					Port:     e.Port,
					Metadata: txt.Metadata,
				})

				serviceMap[txt.Version] = s

			case <-p.Context.Done():
				close(done)
				return
			}
		}
	}()

	// execute the query
	if err := mdns.Query(p); err != nil {
		return nil, err
	}

	// wait for completion
	<-done

	// create list and return
	var services []*Service

	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

func (m *mdnsRegistry) ListServices() ([]*Service, error) {
	serviceMap := make(map[string]bool)
	entries := make(chan *mdns.ServiceEntry, 100)
	done := make(chan bool)

	p := mdns.DefaultParams("_services")
	m.RLock()
	if len(m.domain) > 0 {
		p.Domain = m.domain
	}
	if m.iface != nil {
		p.Interface = m.iface
	}
	timeout := m.opts.Timeout
	m.RUnlock()

	// set context with timeout
	p.Context, _ = context.WithTimeout(context.Background(), timeout)
	// set entries channel
	p.Entries = entries

	var services []*Service

	go func() {
		for {
			select {
			case e := <-entries:
				if e.TTL == 0 {
					continue
				}

				name := strings.TrimSuffix(e.Name, "."+p.Service+"."+p.Domain+".")
				if !serviceMap[name] {
					serviceMap[name] = true
					services = append(services, &Service{Name: name})
				}
			case <-p.Context.Done():
				close(done)
				return
			}
		}
	}()

	// execute query
	if err := mdns.Query(p); err != nil {
		return nil, err
	}

	// wait till done
	<-done

	return services, nil
}

func (m *mdnsRegistry) Watch(opts ...WatchOption) (Watcher, error) {
	var wo WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	m.RLock()
	domain := m.domain
	m.RUnlock()

	md := &mdnsWatcher{
		domain: domain,
		wo:     wo,
		ch:     make(chan *mdns.ServiceEntry, 100),
		exit:   make(chan struct{}),
	}

	go func() {
		if err := mdns.Listen(md.ch, md.exit); err != nil {
			md.Stop()
		}
	}()

	return md, nil
}

func (m *mdnsRegistry) String() string {
	return "mdns"
}

// NewRegistry returns a new default registry which is mdns
func NewRegistry(opts ...Option) Registry {
	return newRegistry(opts...)
}

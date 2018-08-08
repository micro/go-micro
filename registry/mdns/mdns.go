package mdns

/*
	MDNS is a multicast dns registry for service discovery
	This creates a zero dependency system which is great
	where multicast dns is available. This usually depends
	on the ability to leverage udp and multicast/broadcast.
*/

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/registry"
	"github.com/micro/mdns"
	hash "github.com/mitchellh/hashstructure"
)

type mdnsTxt struct {
	Service   string
	Version   string
	Endpoints []*registry.Endpoint
	Metadata  map[string]string
}

type mdnsEntry struct {
	hash uint64
	id   string
	node *mdns.Server
}

type mdnsRegistry struct {
	opts registry.Options

	sync.Mutex
	services map[string][]*mdnsEntry
}

func newRegistry(opts ...registry.Option) registry.Registry {
	options := registry.Options{
		Timeout: time.Millisecond * 100,
	}

	return &mdnsRegistry{
		opts:     options,
		services: make(map[string][]*mdnsEntry),
	}
}

func (m *mdnsRegistry) Init(opts ...registry.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}

func (m *mdnsRegistry) Options() registry.Options {
	return m.opts
}

func (m *mdnsRegistry) Register(service *registry.Service, opts ...registry.RegisterOption) error {
	m.Lock()
	defer m.Unlock()

	entries, ok := m.services[service.Name]
	// first entry, create wildcard used for list queries
	if !ok {
		s, err := mdns.NewMDNSService(
			service.Name,
			"_services",
			"",
			"",
			9999,
			[]net.IP{net.ParseIP("0.0.0.0")},
			nil,
		)
		if err != nil {
			return err
		}

		srv, err := mdns.NewServer(&mdns.Config{Zone: &mdns.DNSSDService{s}})
		if err != nil {
			return err
		}

		// append the wildcard entry
		entries = append(entries, &mdnsEntry{id: "*", node: srv})
	}

	var gerr error

	for _, node := range service.Nodes {
		// create hash of service; uint64
		h, err := hash.Hash(node, nil)
		if err != nil {
			gerr = err
			continue
		}

		var seen bool
		var e *mdnsEntry

		for _, entry := range entries {
			if node.Id == entry.id {
				seen = true
				e = entry
				break
			}
		}

		// already registered, continue
		if seen && e.hash == h {
			continue
			// hash doesn't match, shutdown
		} else if seen {
			e.node.Shutdown()
			// doesn't exist
		} else {
			e = &mdnsEntry{hash: h}
		}

		txt, err := encode(&mdnsTxt{
			Service:   service.Name,
			Version:   service.Version,
			Endpoints: service.Endpoints,
			Metadata:  node.Metadata,
		})

		if err != nil {
			gerr = err
			continue
		}

		// we got here, new node
		s, err := mdns.NewMDNSService(
			node.Id,
			service.Name,
			"",
			"",
			node.Port,
			[]net.IP{net.ParseIP(node.Address)},
			txt,
		)
		if err != nil {
			gerr = err
			continue
		}

		srv, err := mdns.NewServer(&mdns.Config{Zone: s})
		if err != nil {
			gerr = err
			continue
		}

		e.id = node.Id
		e.node = srv
		entries = append(entries, e)
	}

	// save
	m.services[service.Name] = entries

	return gerr
}

func (m *mdnsRegistry) Deregister(service *registry.Service) error {
	m.Lock()
	defer m.Unlock()

	var newEntries []*mdnsEntry

	// loop existing entries, check if any match, shutdown those that do
	for _, entry := range m.services[service.Name] {
		var remove bool

		for _, node := range service.Nodes {
			if node.Id == entry.id {
				entry.node.Shutdown()
				remove = true
				break
			}
		}

		// keep it?
		if !remove {
			newEntries = append(newEntries, entry)
		}
	}

	// last entry is the wildcard for list queries. Remove it.
	if len(newEntries) == 1 && newEntries[0].id == "*" {
		newEntries[0].node.Shutdown()
		delete(m.services, service.Name)
	} else {
		m.services[service.Name] = newEntries
	}

	return nil
}

func (m *mdnsRegistry) GetService(service string) ([]*registry.Service, error) {
	p := mdns.DefaultParams(service)
	p.Timeout = m.opts.Timeout
	entryCh := make(chan *mdns.ServiceEntry, 10)
	p.Entries = entryCh

	exit := make(chan bool)
	defer close(exit)

	serviceMap := make(map[string]*registry.Service)

	go func() {
		for {
			select {
			case e := <-entryCh:
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

				if txt.Service != service {
					continue
				}

				s, ok := serviceMap[txt.Version]
				if !ok {
					s = &registry.Service{
						Name:      txt.Service,
						Version:   txt.Version,
						Endpoints: txt.Endpoints,
					}
				}

				s.Nodes = append(s.Nodes, &registry.Node{
					Id:       strings.TrimSuffix(e.Name, "."+p.Service+"."+p.Domain+"."),
					Address:  e.AddrV4.String(),
					Port:     e.Port,
					Metadata: txt.Metadata,
				})

				serviceMap[txt.Version] = s
			case <-exit:
				return
			}
		}
	}()

	if err := mdns.Query(p); err != nil {
		return nil, err
	}

	// create list and return
	var services []*registry.Service

	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

func (m *mdnsRegistry) ListServices() ([]*registry.Service, error) {
	p := mdns.DefaultParams("_services")
	p.Timeout = m.opts.Timeout
	entryCh := make(chan *mdns.ServiceEntry, 10)
	p.Entries = entryCh

	exit := make(chan bool)
	defer close(exit)

	serviceMap := make(map[string]bool)
	var services []*registry.Service

	go func() {
		for {
			select {
			case e := <-entryCh:
				if e.TTL == 0 {
					continue
				}

				name := strings.TrimSuffix(e.Name, "."+p.Service+"."+p.Domain+".")
				if !serviceMap[name] {
					serviceMap[name] = true
					services = append(services, &registry.Service{Name: name})
				}
			case <-exit:
				return
			}
		}
	}()

	if err := mdns.Query(p); err != nil {
		return nil, err
	}

	return services, nil
}

func (m *mdnsRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	md := &mdnsWatcher{
		wo:   wo,
		ch:   make(chan *mdns.ServiceEntry, 32),
		exit: make(chan struct{}),
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

func NewRegistry(opts ...registry.Option) registry.Registry {
	return newRegistry(opts...)
}

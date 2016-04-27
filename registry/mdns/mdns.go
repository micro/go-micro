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

	"github.com/hashicorp/mdns"
	"github.com/micro/go-micro/registry"
	hash "github.com/mitchellh/hashstructure"
)

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

		srv, err := mdns.NewServer(&mdns.Config{Zone: s})
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

		var txt []string
		txt = append(txt, encodeVersion(service.Version)...)
		txt = append(txt, encodeMetadata(node.Metadata)...)
		//		txt = append(txt, encodeEndpoints(service.Endpoints)...)

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

				version, exists := decodeVersion(e.InfoFields)
				if !exists {
					continue
				}

				s, ok := serviceMap[version]
				if !ok {
					s = &registry.Service{
						Name:    service,
						Version: version,
						//						Endpoints: decodeEndpoints(e.InfoFields),
					}
				}

				s.Nodes = append(s.Nodes, &registry.Node{
					Id:       strings.TrimSuffix(e.Name, "."+p.Service+"."+p.Domain+"."),
					Address:  e.AddrV4.String(),
					Port:     e.Port,
					Metadata: decodeMetadata(e.InfoFields),
				})

				serviceMap[version] = s
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

func (m *mdnsRegistry) Watch() (registry.Watcher, error) {
	return nil, nil
}

func (m *mdnsRegistry) String() string {
	return "mdns"
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	return newRegistry(opts...)
}

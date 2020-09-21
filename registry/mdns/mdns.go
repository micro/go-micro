package mdns

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/registry"
	"github.com/micro/go-micro/v3/util/mdns"
)

const (
	// every service is written to the global domain so * domain queries work, e.g.
	// calling mdns.List(registry.ListDomain("*")) will list the services across all
	// domains
	globalDomain = "global"
)

type mdnsTxt struct {
	Service   string
	Version   string
	Endpoints []*registry.Endpoint
	Metadata  map[string]string
}

type mdnsEntry struct {
	id   string
	node *mdns.Server
}

// services are a key/value map, with the service name as a key and the value being a
// slice of mdns entries, representing the nodes with a single _services entry to be
// used for listing
type services map[string][]*mdnsEntry

// mdsRegistry is a multicast dns registry
type mdnsRegistry struct {
	opts registry.Options

	// the top level domains, these can be overriden using options
	defaultDomain string
	globalDomain  string

	sync.Mutex
	domains map[string]services

	mtx sync.RWMutex

	// watchers
	watchers map[string]*mdnsWatcher

	// listener
	listener chan *mdns.ServiceEntry
}

type mdnsWatcher struct {
	id   string
	wo   registry.WatchOptions
	ch   chan *mdns.ServiceEntry
	exit chan struct{}
	// the mdns domain
	domain string
	// the registry
	registry *mdnsRegistry
}

func encode(txt *mdnsTxt) ([]string, error) {
	b, err := json.Marshal(txt)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	defer buf.Reset()

	w := zlib.NewWriter(&buf)
	defer func() {
		if closeErr := w.Close(); closeErr != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Errorf("[mdns] registry close encoding writer err: %v", closeErr)
			}
		}
	}()
	if _, err := w.Write(b); err != nil {
		return nil, err
	}

	if err = w.Close(); err != nil {
		return nil, err
	}

	encoded := hex.EncodeToString(buf.Bytes())
	// individual txt limit
	if len(encoded) <= 255 {
		return []string{encoded}, nil
	}

	// split encoded string
	var record []string

	for len(encoded) > 255 {
		record = append(record, encoded[:255])
		encoded = encoded[255:]
	}

	record = append(record, encoded)

	return record, nil
}

func decode(record []string) (*mdnsTxt, error) {
	encoded := strings.Join(record, "")

	hr, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	br := bytes.NewReader(hr)
	zr, err := zlib.NewReader(br)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	rbuf, err := ioutil.ReadAll(zr)
	if err != nil {
		return nil, err
	}

	var txt *mdnsTxt

	if err := json.Unmarshal(rbuf, &txt); err != nil {
		return nil, err
	}

	return txt, nil
}

func newRegistry(opts ...registry.Option) registry.Registry {
	options := registry.Options{
		Context: context.Background(),
		Timeout: time.Millisecond * 100,
	}

	for _, o := range opts {
		o(&options)
	}

	// set the domain
	defaultDomain := registry.DefaultDomain
	if d, ok := options.Context.Value("mdns.domain").(string); ok {
		defaultDomain = d
	}

	return &mdnsRegistry{
		defaultDomain: defaultDomain,
		globalDomain:  globalDomain,
		opts:          options,
		domains:       make(map[string]services),
		watchers:      make(map[string]*mdnsWatcher),
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

// createServiceMDNSEntry will create a new wildcard mdns entry for the service in the
// given domain. This wildcard mdns entry is used when listing services.
func createServiceMDNSEntry(name, domain string) (*mdnsEntry, error) {
	ip := net.ParseIP("0.0.0.0")

	s, err := mdns.NewMDNSService(name, "_services", domain+".", "", 9999, []net.IP{ip}, nil)
	if err != nil {
		return nil, err
	}

	srv, err := mdns.NewServer(&mdns.Config{Zone: &mdns.DNSSDService{MDNSService: s}, LocalhostChecking: true})
	if err != nil {
		return nil, err
	}

	return &mdnsEntry{id: "*", node: srv}, nil
}

func (m *mdnsRegistry) createMDNSEntries(domain, serviceName string) ([]*mdnsEntry, error) {
	// if it already exists don't reegister it again
	entries, ok := m.domains[domain][serviceName]
	if ok {
		return entries, nil
	}

	// create the wildcard entry used for list queries in this domain
	entry, err := createServiceMDNSEntry(serviceName, domain)
	if err != nil {
		return nil, err
	}

	return []*mdnsEntry{entry}, nil
}

func registerService(service *registry.Service, entries []*mdnsEntry, options registry.RegisterOptions) ([]*mdnsEntry, error) {
	var lastError error
	for _, node := range service.Nodes {
		var seen bool

		for _, entry := range entries {
			if node.Id == entry.id {
				seen = true
				break
			}
		}

		// this node has already been registered, continue
		if seen {
			continue
		}

		txt, err := encode(&mdnsTxt{
			Service:   service.Name,
			Version:   service.Version,
			Endpoints: service.Endpoints,
			Metadata:  node.Metadata,
		})

		if err != nil {
			lastError = err
			continue
		}

		host, pt, err := net.SplitHostPort(node.Address)
		if err != nil {
			lastError = err
			continue
		}
		port, _ := strconv.Atoi(pt)

		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("[mdns] registry create new service with ip: %s for: %s", net.ParseIP(host).String(), host)
		}
		// we got here, new node
		s, err := mdns.NewMDNSService(
			node.Id,
			service.Name,
			options.Domain+".",
			"",
			port,
			[]net.IP{net.ParseIP(host)},
			txt,
		)
		if err != nil {
			lastError = err
			continue
		}

		srv, err := mdns.NewServer(&mdns.Config{Zone: s, LocalhostChecking: true})
		if err != nil {
			lastError = err
			continue
		}

		entries = append(entries, &mdnsEntry{id: node.Id, node: srv})
	}

	return entries, lastError
}

func createGlobalDomainService(service *registry.Service, options registry.RegisterOptions) *registry.Service {
	srv := *service
	srv.Nodes = nil

	for _, n := range service.Nodes {
		node := n

		// set the original domain in node metadata
		if node.Metadata == nil {
			node.Metadata = map[string]string{"domain": options.Domain}
		} else {
			node.Metadata["domain"] = options.Domain
		}

		srv.Nodes = append(srv.Nodes, node)
	}

	return &srv
}

func (m *mdnsRegistry) Register(service *registry.Service, opts ...registry.RegisterOption) error {
	m.Lock()

	// parse the options
	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.Domain) == 0 {
		options.Domain = m.defaultDomain
	}

	// create the domain in the memory store if it doesn't yet exist
	if _, ok := m.domains[options.Domain]; !ok {
		m.domains[options.Domain] = make(services)
	}

	entries, err := m.createMDNSEntries(options.Domain, service.Name)
	if err != nil {
		m.Unlock()
		return err
	}

	entries, gerr := registerService(service, entries, options)

	// save the mdns entry
	m.domains[options.Domain][service.Name] = entries
	m.Unlock()

	// register in the global Domain so it can be queried as one
	if options.Domain != m.globalDomain {
		srv := createGlobalDomainService(service, options)
		if err := m.Register(srv, append(opts, registry.RegisterDomain(m.globalDomain))...); err != nil {
			gerr = err
		}
	}

	return gerr
}

func (m *mdnsRegistry) Deregister(service *registry.Service, opts ...registry.DeregisterOption) error {
	// parse the options
	var options registry.DeregisterOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.Domain) == 0 {
		options.Domain = m.defaultDomain
	}

	// register in the global Domain
	var err error
	if options.Domain != m.globalDomain {
		defer func() {
			err = m.Deregister(service, append(opts, registry.DeregisterDomain(m.globalDomain))...)
		}()
	}

	// we want to unlock before we call deregister on the global domain, so it's important this unlock
	// is applied after the defer m.Deregister is called above
	m.Lock()
	defer m.Unlock()

	// the service wasn't registered, we can safely exist
	if _, ok := m.domains[options.Domain]; !ok {
		return err
	}

	// loop existing entries, check if any match, shutdown those that do
	var newEntries []*mdnsEntry
	for _, entry := range m.domains[options.Domain][service.Name] {
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

	// we have no new entries, we can exit
	if len(newEntries) == 0 {
		return nil
	}

	// we have more than one entry remaining, we can exit
	if len(newEntries) > 1 {
		m.domains[options.Domain][service.Name] = newEntries
		return err
	}

	// our remaining entry is not a wildcard, we can exit
	if len(newEntries) == 1 && newEntries[0].id != "*" {
		m.domains[options.Domain][service.Name] = newEntries
		return err
	}

	// last entry is the wildcard for list queries. Remove it.
	newEntries[0].node.Shutdown()
	delete(m.domains[options.Domain], service.Name)

	// check to see if we can delete the domain entry
	if len(m.domains[options.Domain]) == 0 {
		delete(m.domains, options.Domain)
	}

	return err
}

func (m *mdnsRegistry) GetService(service string, opts ...registry.GetOption) ([]*registry.Service, error) {
	// parse the options
	var options registry.GetOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.Domain) == 0 {
		options.Domain = m.defaultDomain
	}
	if options.Domain == registry.WildcardDomain {
		options.Domain = m.globalDomain
	}

	serviceMap := make(map[string]*registry.Service)
	entries := make(chan *mdns.ServiceEntry, 10)
	done := make(chan bool)

	p := mdns.DefaultParams(service)
	// set context with timeout
	var cancel context.CancelFunc
	p.Context, cancel = context.WithTimeout(context.Background(), m.opts.Timeout)
	defer cancel()
	// set entries channel
	p.Entries = entries
	// set the domain
	p.Domain = options.Domain

	go func() {
		for {
			select {
			case e := <-entries:
				// list record so skip
				if e.Name == "_services" {
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
				addr := ""
				// prefer ipv4 addrs
				if len(e.AddrV4) > 0 {
					addr = e.AddrV4.String()
					// else use ipv6
				} else if len(e.AddrV6) > 0 {
					addr = "[" + e.AddrV6.String() + "]"
				} else {
					if logger.V(logger.InfoLevel, logger.DefaultLogger) {
						logger.Infof("[mdns]: invalid endpoint received: %v", e)
					}
					continue
				}
				s.Nodes = append(s.Nodes, &registry.Node{
					Id:       strings.TrimSuffix(e.Name, "."+p.Service+"."+p.Domain+"."),
					Address:  fmt.Sprintf("%s:%d", addr, e.Port),
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
	services := make([]*registry.Service, 0, len(serviceMap))

	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

func (m *mdnsRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	// parse the options
	var options registry.ListOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.Domain) == 0 {
		options.Domain = m.defaultDomain
	}
	if options.Domain == registry.WildcardDomain {
		options.Domain = m.globalDomain
	}

	serviceMap := make(map[string]bool)
	entries := make(chan *mdns.ServiceEntry, 10)
	done := make(chan bool)

	p := mdns.DefaultParams("_services")
	// set context with timeout
	var cancel context.CancelFunc
	p.Context, cancel = context.WithTimeout(context.Background(), m.opts.Timeout)
	defer cancel()
	// set entries channel
	p.Entries = entries
	// set domain
	p.Domain = options.Domain

	var services []*registry.Service

	go func() {
		for {
			select {
			case e := <-entries:
				if e.TTL == 0 {
					continue
				}
				if !strings.HasSuffix(e.Name, p.Domain+".") {
					continue
				}
				name := strings.TrimSuffix(e.Name, "."+p.Service+"."+p.Domain+".")
				if !serviceMap[name] {
					serviceMap[name] = true
					services = append(services, &registry.Service{Name: name})
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

func (m *mdnsRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}
	if len(wo.Domain) == 0 {
		wo.Domain = m.defaultDomain
	}
	if wo.Domain == registry.WildcardDomain {
		wo.Domain = m.globalDomain
	}

	md := &mdnsWatcher{
		id:       uuid.New().String(),
		wo:       wo,
		ch:       make(chan *mdns.ServiceEntry, 32),
		exit:     make(chan struct{}),
		domain:   wo.Domain,
		registry: m,
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	// save the watcher
	m.watchers[md.id] = md

	// check of the listener exists
	if m.listener != nil {
		return md, nil
	}

	// start the listener
	go func() {
		// go to infinity
		for {
			m.mtx.Lock()

			// just return if there are no watchers
			if len(m.watchers) == 0 {
				m.listener = nil
				m.mtx.Unlock()
				return
			}

			// check existing listener
			if m.listener != nil {
				m.mtx.Unlock()
				return
			}

			// reset the listener
			exit := make(chan struct{})
			ch := make(chan *mdns.ServiceEntry, 32)
			m.listener = ch

			m.mtx.Unlock()

			// send messages to the watchers
			go func() {
				send := func(w *mdnsWatcher, e *mdns.ServiceEntry) {
					select {
					case w.ch <- e:
					default:
					}
				}

				for {
					select {
					case <-exit:
						return
					case e, ok := <-ch:
						if !ok {
							return
						}
						m.mtx.RLock()
						// send service entry to all watchers
						for _, w := range m.watchers {
							send(w, e)
						}
						m.mtx.RUnlock()
					}
				}

			}()

			// start listening, blocking call
			mdns.Listen(ch, exit)

			// mdns.Listen has unblocked
			// kill the saved listener
			m.mtx.Lock()
			m.listener = nil
			close(ch)
			m.mtx.Unlock()
		}
	}()

	return md, nil
}

func (m *mdnsRegistry) String() string {
	return "mdns"
}

func (m *mdnsWatcher) Next() (*registry.Result, error) {
	for {
		select {
		case e := <-m.ch:
			txt, err := decode(e.InfoFields)
			if err != nil {
				continue
			}

			if len(txt.Service) == 0 || len(txt.Version) == 0 {
				continue
			}

			// Filter watch options
			// wo.Service: Only keep services we care about
			if len(m.wo.Service) > 0 && txt.Service != m.wo.Service {
				continue
			}
			var action string
			if e.TTL == 0 {
				action = "delete"
			} else {
				action = "create"
			}

			service := &registry.Service{
				Name:      txt.Service,
				Version:   txt.Version,
				Endpoints: txt.Endpoints,
				Metadata:  txt.Metadata,
			}

			// skip anything without the domain we care about
			suffix := fmt.Sprintf(".%s.%s.", service.Name, m.domain)
			if !strings.HasSuffix(e.Name, suffix) {
				continue
			}

			var addr string
			if len(e.AddrV4) > 0 {
				addr = e.AddrV4.String()
			} else if len(e.AddrV6) > 0 {
				addr = "[" + e.AddrV6.String() + "]"
			} else {
				addr = e.Addr.String()
			}

			service.Nodes = append(service.Nodes, &registry.Node{
				Id:       strings.TrimSuffix(e.Name, suffix),
				Address:  fmt.Sprintf("%s:%d", addr, e.Port),
				Metadata: txt.Metadata,
			})

			return &registry.Result{
				Action:  action,
				Service: service,
			}, nil
		case <-m.exit:
			return nil, registry.ErrWatcherStopped
		}
	}
}

func (m *mdnsWatcher) Stop() {
	select {
	case <-m.exit:
		return
	default:
		close(m.exit)
		// remove self from the registry
		m.registry.mtx.Lock()
		delete(m.registry.watchers, m.id)
		m.registry.mtx.Unlock()
	}
}

// NewRegistry returns a new default registry which is mdns
func NewRegistry(opts ...registry.Option) registry.Registry {
	return newRegistry(opts...)
}

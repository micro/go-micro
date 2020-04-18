// Package multicast is a multicast registry
package multicast

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/registry"
	regsvc "github.com/micro/go-micro/v2/registry/service"
	regpb "github.com/micro/go-micro/v2/registry/service/proto"
	regutil "github.com/micro/go-micro/v2/util/registry"
	"github.com/oxtoacart/bpool"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// create buffer pool with 16 instances each preallocated with 256 bytes
var bufferPool = bpool.NewSizedBufferPool(16, 256)

/*
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
*/

type mcastRegistry struct {
	opts registry.Options
	sync.RWMutex

	services map[string]*registry.Service
	// watchers
	//watchers map[string]*mdnsWatcher

	gr4    net.IP
	gr6    net.IP
	ifaces []net.Interface
	conn4  *ipv4.PacketConn
	conn6  *ipv6.PacketConn
}

type mcastWatcher struct {
	id string
	wo registry.WatchOptions
	//ch   chan *mdns.ServiceEntry
	exit chan struct{}
	// the mdns domain
	//	domain string
	// the registry
	//registry *mdnsRegistry
}

/*
func encode(txt *mdnsTxt) ([]string, error) {
	b, err := json.Marshal(txt)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	defer buf.Reset()

	w := zlib.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		return nil, err
	}
	w.Close()

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
*/

func newRegistry(opts ...registry.Option) registry.Registry {
	options := registry.Options{
		Context: context.Background(),
		Timeout: time.Millisecond * 100,
	}

	for _, o := range opts {
		o(&options)
	}

	reg := &mcastRegistry{
		opts:     options,
		services: make(map[string]*registry.Service),
		//		watchers: make(map[string]*mdnsWatcher),
	}

	return reg
}

func (m *mcastRegistry) Init(opts ...registry.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
		return err
	}

	ln6, err := net.ListenPacket("udp6", "[ff02::]:65353")
	if err != nil {
		panic(err)
		return err
	}
	gr6 := net.ParseIP("ff02::fb")
	conn6 := ipv6.NewPacketConn(ln6)
	for _, iface := range ifaces {
		logger.Infof("up:%v mc:%v", iface.Flags&net.FlagUp, iface.Flags&net.FlagMulticast)
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		if err := conn6.JoinGroup(&iface, &net.UDPAddr{IP: gr6}); err != nil {
			continue
			panic(err)
			return err
		}
	}
	if err := conn6.SetControlMessage(ipv6.FlagDst, true); err != nil {
		panic(err)
		return err
	}
	//conn6.SetTOS(0x0)
	//conn6.SetTTL(16)
	if err = conn6.SetMulticastLoopback(true); err != nil {
		return err
	}

	ln4, err := net.ListenPacket("udp4", "224.0.0.251:65353")
	if err != nil {
		panic(err)
		return err
	}
	gr4 := net.ParseIP("224.0.0.251")
	conn4 := ipv4.NewPacketConn(ln4)
	for _, iface := range ifaces {
		logger.Infof("up:%v mc:%v", iface.Flags&net.FlagUp, iface.Flags&net.FlagMulticast)
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		if err := conn4.JoinGroup(&iface, &net.UDPAddr{IP: gr4}); err != nil {
			continue
			logger.Infof("%#+v\n", iface)
			panic(err)
			return err
		}
	}
	if err := conn4.SetControlMessage(ipv4.FlagDst, true); err != nil {
		panic(err)
		return err
	}
	if err = conn4.SetMulticastLoopback(true); err != nil {
		return err
	}

	//conn4.SetTOS(0x0)
	//conn4.SetTTL(16)

	m.Lock()
	m.gr4 = gr4
	m.conn4 = conn4
	m.gr6 = gr6
	m.conn6 = conn6
	m.ifaces = ifaces
	m.Unlock()

	go func() {
		// ethernet default packet size
		buf := make([]byte, 1500)
		for {
			select {
			default:
				n, cm, src, err := conn6.ReadFrom(buf)
				if err != nil {
					logger.Info(err)
					continue
				}
				if cm.Dst.IsMulticast() {
					if cm.Dst.Equal(gr6) {
						pbuf := proto.NewBuffer(buf[:n])
						pb := &regpb.Service{}
						err := pbuf.Unmarshal(pb)
						pbuf.Reset()
						if err != nil {
							// error continue
							continue
						}
						service := regsvc.ToService(pb)
						fmt.Printf("ip6 %#+v %#+v\n", src, service)
					} else {
						// unknown group, discard
						continue
					}
				}
			}
		}
	}()

	go func() {
		// ethernet default packet size
		buf := make([]byte, 1500)
		for {
			select {
			default:
				n, cm, src, err := conn4.ReadFrom(buf)
				if err != nil {
					logger.Info(err)
					continue
				}
				if cm.Dst.IsMulticast() {
					if cm.Dst.Equal(gr4) {
						pbuf := proto.NewBuffer(buf[:n])
						pb := &regpb.Service{}
						err := pbuf.Unmarshal(pb)
						pbuf.Reset()
						if err != nil {
							// error continue
							continue
						}
						service := regsvc.ToService(pb)
						fmt.Printf("ip4 %#+v %#+v\n", src, service)
					} else {
						// unknown group, discard
						continue
					}
				}
			}
		}
	}()

	return nil
}

func (m *mcastRegistry) Options() registry.Options {
	return m.opts
}

func (m *mcastRegistry) Register(service *registry.Service, opts ...registry.RegisterOption) error {
	m.Lock()
	_, ok := m.services[service.Name]
	if !ok {
		entries := regutil.CopyService(service)
		m.services[service.Name] = entries
	}
	m.Unlock()

	pb := regsvc.ToProto(service)

	buf := bufferPool.Get()
	pbuf := proto.NewBuffer(buf.Bytes())
	defer func() {
		bufferPool.Put(bytes.NewBuffer(pbuf.Bytes()))
	}()

	if err := pbuf.Marshal(pb); err != nil {
		return err
	}

	logger.Info("LOCK")
	m.RLock()
	gr4 := m.gr4
	conn4 := m.conn4
	gr6 := m.gr6
	conn6 := m.conn6
	ifaces := m.ifaces
	m.RUnlock()

	cm4 := &ipv4.ControlMessage{}
	cm6 := &ipv6.ControlMessage{}
	udp4 := &net.UDPAddr{IP: gr4, Port: 65353}
	udp6 := &net.UDPAddr{IP: gr6, Port: 65353}

	for _, iface := range ifaces {
		cm4.IfIndex = iface.Index
		cm6.IfIndex = iface.Index
		if _, err := conn4.WriteTo(pbuf.Bytes(), cm4, udp4); err != nil {
			logger.Info(err)
		}
		if _, err := conn6.WriteTo(pbuf.Bytes(), cm6, udp6); err != nil {
			logger.Info(err)
		}
	}

	return nil
}

func (m *mcastRegistry) Deregister(service *registry.Service, opts ...registry.DeregisterOption) error {
	m.Lock()
	defer m.Unlock()

	// loop existing entries, check if any match, shutdown those that do
	entries, ok := m.services[service.Name]
	if !ok {
		return registry.ErrNotFound
	}
	logger.Infof("entries: %#+v\n", entries)
	logger.Info("deregister send multicast")
	/*
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
	*/
	return nil
}

func (m *mcastRegistry) GetService(service string, opts ...registry.GetOption) ([]*registry.Service, error) {
	/*
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
		p.Domain = m.domain

		go func() {
			for {
				select {
				case e := <-entries:
					// list record so skip
					if p.Service == "_services" {
						continue
					}
					if p.Domain != m.domain {
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
					if e.AddrV4 != nil {
						addr = e.AddrV4.String()
						// else use ipv6
					} else if e.AddrV6 != nil {
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
	*/

	return nil, registry.ErrNotFound
}

func (m *mcastRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	return nil, nil
	/*
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
		p.Domain = m.domain

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
	*/
}

func (m *mcastRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}
	/*
		md := &mdnsWatcher{
			id:       uuid.New().String(),
			wo:       wo,
			ch:       make(chan *mdns.ServiceEntry, 32),
			exit:     make(chan struct{}),
			domain:   m.domain,
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
	*/
	return nil, nil
}

func (m *mcastRegistry) String() string {
	return "multicast"
}

func (m *mcastWatcher) Next() (*registry.Result, error) {
	return nil, registry.ErrWatcherStopped
	/*
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
				}

				// skip anything without the domain we care about
				suffix := fmt.Sprintf(".%s.%s.", service.Name, m.domain)
				if !strings.HasSuffix(e.Name, suffix) {
					continue
				}

				service.Nodes = append(service.Nodes, &registry.Node{
					Id:       strings.TrimSuffix(e.Name, suffix),
					Address:  fmt.Sprintf("%s:%d", e.AddrV4.String(), e.Port),
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
	*/
}

func (m *mcastWatcher) Stop() {
	/*
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
	*/
}

// NewRegistry returns a new default registry which is mdns
func NewRegistry(opts ...registry.Option) registry.Registry {
	return newRegistry(opts...)
}

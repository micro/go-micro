package registry

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	consul "github.com/hashicorp/consul/api"
	hash "github.com/mitchellh/hashstructure"
)

type consulRegistry struct {
	Address string
	Client  *consul.Client
	Options Options

	sync.Mutex
	register map[string]uint64
}

func newTransport(config *tls.Config) *http.Transport {
	if config == nil {
		config = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     config,
	}
	runtime.SetFinalizer(&t, func(tr **http.Transport) {
		(*tr).CloseIdleConnections()
	})
	return t
}

func newConsulRegistry(opts ...Option) Registry {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	// use default config
	config := consul.DefaultConfig()
	if options.Context != nil {
		// Use the consul config passed in the options, if available
		if c, ok := options.Context.Value("consul_config").(*consul.Config); ok {
			config = c
		}
	}
	if config.HttpClient == nil {
		config.HttpClient = new(http.Client)
	}

	// set timeout
	if options.Timeout > 0 {
		config.HttpClient.Timeout = options.Timeout
	}

	// check if there are any addrs
	if len(options.Addrs) > 0 {
		addr, port, err := net.SplitHostPort(options.Addrs[0])
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "8500"
			addr = options.Addrs[0]
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		} else if err == nil {
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		}
	}

	// requires secure connection?
	if options.Secure || options.TLSConfig != nil {
		config.Scheme = "https"
		// We're going to support InsecureSkipVerify
		config.HttpClient.Transport = newTransport(options.TLSConfig)
	}

	// create the client
	client, _ := consul.NewClient(config)

	cr := &consulRegistry{
		Address:  config.Address,
		Client:   client,
		Options:  options,
		register: make(map[string]uint64),
	}

	return cr
}

func (c *consulRegistry) Deregister(s *Service) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	// delete our hash of the service
	c.Lock()
	delete(c.register, s.Name)
	c.Unlock()

	node := s.Nodes[0]
	return c.Client.Agent().ServiceDeregister(node.Id)
}

func (c *consulRegistry) Register(s *Service, opts ...RegisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	var options RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	// create hash of service; uint64
	h, err := hash.Hash(s, nil)
	if err != nil {
		return err
	}

	// use first node
	node := s.Nodes[0]

	// get existing hash
	c.Lock()
	v, ok := c.register[s.Name]
	c.Unlock()

	// if it's already registered and matches then just pass the check
	if ok && v == h {
		// if the err is nil we're all good, bail out
		// if not, we don't know what the state is, so full re-register
		if err := c.Client.Agent().PassTTL("service:"+node.Id, ""); err == nil {
			return nil
		}
	}

	// encode the tags
	tags := encodeMetadata(node.Metadata)
	tags = append(tags, encodeEndpoints(s.Endpoints)...)
	tags = append(tags, encodeVersion(s.Version)...)

	var check *consul.AgentServiceCheck

	// if the TTL is greater than 0 create an associated check
	if options.TTL > time.Duration(0) {
		// splay slightly for the watcher?
		splay := time.Second * 5
		deregTTL := options.TTL + splay
		// consul has a minimum timeout on deregistration of 1 minute.
		if options.TTL < time.Minute {
			deregTTL = time.Minute + splay
		}

		check = &consul.AgentServiceCheck{
			TTL: fmt.Sprintf("%v", options.TTL),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%v", deregTTL),
		}
	}

	// register the service
	if err := c.Client.Agent().ServiceRegister(&consul.AgentServiceRegistration{
		ID:      node.Id,
		Name:    s.Name,
		Tags:    tags,
		Port:    node.Port,
		Address: node.Address,
		Check:   check,
	}); err != nil {
		return err
	}

	// save our hash of the service
	c.Lock()
	c.register[s.Name] = h
	c.Unlock()

	// if the TTL is 0 we don't mess with the checks
	if options.TTL == time.Duration(0) {
		return nil
	}

	// pass the healthcheck
	return c.Client.Agent().PassTTL("service:"+node.Id, "")
}

func (c *consulRegistry) GetService(name string) ([]*Service, error) {
	rsp, _, err := c.Client.Health().Service(name, "", false, nil)
	if err != nil {
		return nil, err
	}

	serviceMap := map[string]*Service{}

	for _, s := range rsp {
		if s.Service.Service != name {
			continue
		}

		// version is now a tag
		version, found := decodeVersion(s.Service.Tags)
		// service ID is now the node id
		id := s.Service.ID
		// key is always the version
		key := version
		// address is service address
		address := s.Service.Address

		// if we can't get the version we bail
		// use old the old ways
		if !found {
			continue
		}

		svc, ok := serviceMap[key]
		if !ok {
			svc = &Service{
				Endpoints: decodeEndpoints(s.Service.Tags),
				Name:      s.Service.Service,
				Version:   version,
			}
			serviceMap[key] = svc
		}

		var del bool
		for _, check := range s.Checks {
			// delete the node if the status is critical
			if check.Status == "critical" {
				del = true
				break
			}
		}

		// if delete then skip the node
		if del {
			continue
		}

		svc.Nodes = append(svc.Nodes, &Node{
			Id:       id,
			Address:  address,
			Port:     s.Service.Port,
			Metadata: decodeMetadata(s.Service.Tags),
		})
	}

	var services []*Service
	for _, service := range serviceMap {
		services = append(services, service)
	}
	return services, nil
}

func (c *consulRegistry) ListServices() ([]*Service, error) {
	rsp, _, err := c.Client.Catalog().Services(nil)
	if err != nil {
		return nil, err
	}

	var services []*Service

	for service := range rsp {
		services = append(services, &Service{Name: service})
	}

	return services, nil
}

func (c *consulRegistry) Watch() (Watcher, error) {
	return newConsulWatcher(c)
}

func (c *consulRegistry) String() string {
	return "consul"
}

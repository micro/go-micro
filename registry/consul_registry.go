package registry

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"time"

	consul "github.com/hashicorp/consul/api"
)

type consulRegistry struct {
	Address string
	Client  *consul.Client
	Options Options
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

func encodeEndpoints(en []*Endpoint) []string {
	var tags []string
	for _, e := range en {
		if b, err := json.Marshal(e); err == nil {
			tags = append(tags, "e="+string(b))
		}
	}
	return tags
}

func decodeEndpoints(tags []string) []*Endpoint {
	var en []*Endpoint
	for _, tag := range tags {
		if len(tag) == 0 || tag[0] != 'e' {
			continue
		}

		var e *Endpoint
		if err := json.Unmarshal([]byte(tag[2:]), &e); err == nil {
			en = append(en, e)
		}
	}
	return en
}

func encodeMetadata(md map[string]string) []string {
	var tags []string
	for k, v := range md {
		if b, err := json.Marshal(map[string]string{
			k: v,
		}); err == nil {
			tags = append(tags, "t="+string(b))
		}
	}
	return tags
}

func decodeMetadata(tags []string) map[string]string {
	md := make(map[string]string)
	for _, tag := range tags {
		if len(tag) == 0 || tag[0] != 't' {
			continue
		}

		var kv map[string]string
		if err := json.Unmarshal([]byte(tag[2:]), &kv); err == nil {
			for k, v := range kv {
				md[k] = v
			}
		}
	}
	return md
}

func encodeVersion(v string) string {
	return "v=" + v
}

func decodeVersion(tags []string) (string, bool) {
	for _, tag := range tags {
		if len(tag) == 0 || tag[0] != 'v' {
			continue
		}
		return tag[2:], true
	}
	return "", false
}

func newConsulRegistry(addrs []string, opts ...Option) Registry {
	var opt Options
	for _, o := range opts {
		o(&opt)
	}

	// use default config
	config := consul.DefaultConfig()

	// set timeout
	if opt.Timeout > 0 {
		config.HttpClient.Timeout = opt.Timeout
	}

	// check if there are any addrs
	if len(addrs) > 0 {
		addr, port, err := net.SplitHostPort(addrs[0])
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "8500"
			addr = addrs[0]
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		} else if err == nil {
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		}
	}

	// requires secure connection?
	if opt.Secure || opt.TLSConfig != nil {
		config.Scheme = "https"
		// We're going to support InsecureSkipVerify
		config.HttpClient.Transport = newTransport(opt.TLSConfig)
	}

	// create the client
	client, _ := consul.NewClient(config)

	cr := &consulRegistry{
		Address: config.Address,
		Client:  client,
		Options: opt,
	}

	return cr
}

func (c *consulRegistry) Deregister(s *Service) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

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

	node := s.Nodes[0]

	tags := encodeMetadata(node.Metadata)
	tags = append(tags, encodeEndpoints(s.Endpoints)...)
	tags = append(tags, encodeVersion(s.Version))

	if err := c.Client.Agent().ServiceRegister(&consul.AgentServiceRegistration{
		ID:      node.Id,
		Name:    s.Name,
		Tags:    tags,
		Port:    node.Port,
		Address: node.Address,
		Check: &consul.AgentServiceCheck{
			TTL: fmt.Sprintf("%v", options.TTL),
		},
	}); err != nil {
		return err
	}

	return c.Client.Agent().PassTTL("service:"+node.Id, "")
}

func (c *consulRegistry) GetService(name string) ([]*Service, error) {
	rsp, _, err := c.Client.Health().Service(name, "", true, nil)
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

		// if we can't get the new type of version
		// use old the old ways
		if !found {
			// id was set as node
			id = s.Node.Node
			// key was service id
			key = s.Service.ID
			// version was service id
			version = s.Service.ID
			// address was address
			address = s.Node.Address
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

	for service, _ := range rsp {
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

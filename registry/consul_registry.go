package registry

import (
	"encoding/json"
	"errors"
	"sync"

	consul "github.com/hashicorp/consul/api"
)

type consulRegistry struct {
	Address string
	Client  *consul.Client

	mtx      sync.RWMutex
	services map[string]*Service
}

func encodeMetadata(md map[string]string) []string {
	var tags []string
	for k, v := range md {
		if b, err := json.Marshal(map[string]string{
			k: v,
		}); err == nil {
			tags = append(tags, string(b))
		}
	}
	return tags
}

func decodeMetadata(tags []string) map[string]string {
	md := make(map[string]string)
	for _, tag := range tags {
		var kv map[string]string
		if err := json.Unmarshal([]byte(tag), &kv); err == nil {
			for k, v := range kv {
				md[k] = v
			}
		}
	}
	return md
}

func newConsulRegistry(addrs []string, opts ...Option) Registry {
	config := consul.DefaultConfig()
	client, _ := consul.NewClient(config)
	if len(addrs) > 0 {
		config.Address = addrs[0]
	}

	cr := &consulRegistry{
		Address:  config.Address,
		Client:   client,
		services: make(map[string]*Service),
	}

	return cr
}

func (c *consulRegistry) Deregister(s *Service) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	node := s.Nodes[0]

	_, err := c.Client.Catalog().Deregister(&consul.CatalogDeregistration{
		Node:      node.Id,
		Address:   node.Address,
		ServiceID: node.Id,
	}, nil)

	return err
}

func (c *consulRegistry) Register(s *Service) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	node := s.Nodes[0]

	tags := encodeMetadata(node.Metadata)

	_, err := c.Client.Catalog().Register(&consul.CatalogRegistration{
		Node:    node.Id,
		Address: node.Address,
		Service: &consul.AgentService{
			ID:      node.Id,
			Service: s.Name,
			Port:    node.Port,
			Tags:    tags,
		},
	}, nil)

	return err
}

func (c *consulRegistry) GetService(name string) (*Service, error) {
	c.mtx.RLock()
	service, ok := c.services[name]
	c.mtx.RUnlock()

	if ok {
		return service, nil
	}

	rsp, _, err := c.Client.Catalog().Service(name, "", nil)
	if err != nil {
		return nil, err
	}

	cs := &Service{}

	for _, s := range rsp {
		if s.ServiceName != name {
			continue
		}

		cs.Name = s.ServiceName
		cs.Nodes = append(cs.Nodes, &Node{
			Id:       s.ServiceID,
			Address:  s.Address,
			Port:     s.ServicePort,
			Metadata: decodeMetadata(s.ServiceTags),
		})
	}

	return cs, nil
}

func (c *consulRegistry) ListServices() ([]*Service, error) {
	c.mtx.RLock()
	serviceMap := c.services
	c.mtx.RUnlock()

	var services []*Service

	if len(serviceMap) > 0 {
		for _, service := range services {
			services = append(services, service)
		}
		return services, nil
	}

	rsp, _, err := c.Client.Catalog().Services(&consul.QueryOptions{})
	if err != nil {
		return nil, err
	}

	for service, _ := range rsp {
		services = append(services, &Service{Name: service})
	}

	return services, nil
}

func (c *consulRegistry) Watch() (Watcher, error) {
	return newConsulWatcher(c)
}

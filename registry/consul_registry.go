package registry

import (
	"errors"
	"sync"

	consul "github.com/hashicorp/consul/api"
)

type ConsulRegistry struct {
	Address string
	Client  *consul.Client

	mtx      sync.RWMutex
	services map[string]Service
}

func (c *ConsulRegistry) Deregister(s Service) error {
	if len(s.Nodes()) == 0 {
		return errors.New("Require at least one node")
	}

	node := s.Nodes()[0]

	_, err := c.Client.Catalog().Deregister(&consul.CatalogDeregistration{
		Node:      node.Id(),
		Address:   node.Address(),
		ServiceID: node.Id(),
	}, nil)

	return err
}

func (c *ConsulRegistry) Register(s Service) error {
	if len(s.Nodes()) == 0 {
		return errors.New("Require at least one node")
	}

	node := s.Nodes()[0]

	_, err := c.Client.Catalog().Register(&consul.CatalogRegistration{
		Node:    node.Id(),
		Address: node.Address(),
		Service: &consul.AgentService{
			ID:      node.Id(),
			Service: s.Name(),
			Port:    node.Port(),
		},
	}, nil)

	return err
}

func (c *ConsulRegistry) GetService(name string) (Service, error) {
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

	cs := &ConsulService{}

	for _, s := range rsp {
		if s.ServiceName != name {
			continue
		}

		cs.ServiceName = s.ServiceName
		cs.ServiceNodes = append(cs.ServiceNodes, &ConsulNode{
			Node:        s.Node,
			NodeId:      s.ServiceID,
			NodeAddress: s.Address,
			NodePort:    s.ServicePort,
		})
	}

	return cs, nil
}

func (c *ConsulRegistry) ListServices() ([]Service, error) {
	c.mtx.RLock()
	serviceMap := c.services
	c.mtx.RUnlock()

	var services []Service

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
		services = append(services, &ConsulService{ServiceName: service})
	}

	return services, nil
}

func (c *ConsulRegistry) NewService(name string, nodes ...Node) Service {
	var snodes []*ConsulNode

	for _, node := range nodes {
		if n, ok := node.(*ConsulNode); ok {
			snodes = append(snodes, n)
		}
	}

	return &ConsulService{
		ServiceName:  name,
		ServiceNodes: snodes,
	}
}

func (c *ConsulRegistry) NewNode(id, address string, port int) Node {
	return &ConsulNode{
		Node:        id,
		NodeId:      id,
		NodeAddress: address,
		NodePort:    port,
	}
}

func (c *ConsulRegistry) Watch() {
	NewConsulWatcher(c)
}

func NewConsulRegistry(addrs []string, opts ...Options) Registry {
	config := consul.DefaultConfig()
	client, _ := consul.NewClient(config)
	if len(addrs) > 0 {
		config.Address = addrs[0]
	}

	cr := &ConsulRegistry{
		Address:  config.Address,
		Client:   client,
		services: make(map[string]Service),
	}

	cr.Watch()
	return cr
}

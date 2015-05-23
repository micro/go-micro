package registry

import (
	"errors"
	"sync"

	consul "github.com/hashicorp/consul/api"
)

type consulRegistry struct {
	Address string
	Client  *consul.Client

	mtx      sync.RWMutex
	services map[string]Service
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
		services: make(map[string]Service),
	}

	cr.Watch()
	return cr
}

func (c *consulRegistry) Deregister(s Service) error {
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

func (c *consulRegistry) Register(s Service) error {
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

func (c *consulRegistry) GetService(name string) (Service, error) {
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

	cs := &consulService{}

	for _, s := range rsp {
		if s.ServiceName != name {
			continue
		}

		cs.ServiceName = s.ServiceName
		cs.ServiceNodes = append(cs.ServiceNodes, &consulNode{
			Node:        s.Node,
			NodeId:      s.ServiceID,
			NodeAddress: s.Address,
			NodePort:    s.ServicePort,
		})
	}

	return cs, nil
}

func (c *consulRegistry) ListServices() ([]Service, error) {
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
		services = append(services, &consulService{ServiceName: service})
	}

	return services, nil
}

func (c *consulRegistry) NewService(name string, nodes ...Node) Service {
	var snodes []*consulNode

	for _, node := range nodes {
		if n, ok := node.(*consulNode); ok {
			snodes = append(snodes, n)
		}
	}

	return &consulService{
		ServiceName:  name,
		ServiceNodes: snodes,
	}
}

func (c *consulRegistry) NewNode(id, address string, port int) Node {
	return &consulNode{
		Node:        id,
		NodeId:      id,
		NodeAddress: address,
		NodePort:    port,
	}
}

func (c *consulRegistry) Watch() {
	newConsulWatcher(c)
}

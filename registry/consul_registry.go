package registry

import (
	"errors"

	consul "github.com/armon/consul-api"
)

type ConsulRegistry struct {
	Client *consul.Client
}

var (
	ConsulCheckTTL = "30s"
)

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

func NewConsulRegistry() Registry {
	client, _ := consul.NewClient(&consul.Config{})

	return &ConsulRegistry{
		Client: client,
	}
}

package registry

type ConsulService struct {
	ServiceName  string
	ServiceNodes []*ConsulNode
}

func (c *ConsulService) Name() string {
	return c.ServiceName
}

func (c *ConsulService) Nodes() []Node {
	var nodes []Node

	for _, node := range c.ServiceNodes {
		nodes = append(nodes, node)
	}

	return nodes
}

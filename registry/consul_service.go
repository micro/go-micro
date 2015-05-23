package registry

type consulService struct {
	ServiceName  string
	ServiceNodes []*consulNode
}

func (c *consulService) Name() string {
	return c.ServiceName
}

func (c *consulService) Nodes() []Node {
	var nodes []Node

	for _, node := range c.ServiceNodes {
		nodes = append(nodes, node)
	}

	return nodes
}

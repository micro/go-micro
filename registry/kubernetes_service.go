package registry

type KubernetesService struct {
	ServiceName  string
	ServiceNodes []*KubernetesNode
}

func (c *KubernetesService) Name() string {
	return c.ServiceName
}

func (c *KubernetesService) Nodes() []Node {
	var nodes []Node

	for _, node := range c.ServiceNodes {
		nodes = append(nodes, node)
	}

	return nodes
}

package registry

type ConsulNode struct {
	Node        string
	NodeId      string
	NodeAddress string
	NodePort    int
}

func (c *ConsulNode) Id() string {
	return c.NodeId
}

func (c *ConsulNode) Address() string {
	return c.NodeAddress
}

func (c *ConsulNode) Port() int {
	return c.NodePort
}

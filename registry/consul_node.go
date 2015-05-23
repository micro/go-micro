package registry

type consulNode struct {
	Node        string
	NodeId      string
	NodeAddress string
	NodePort    int
}

func (c *consulNode) Id() string {
	return c.NodeId
}

func (c *consulNode) Address() string {
	return c.NodeAddress
}

func (c *consulNode) Port() int {
	return c.NodePort
}

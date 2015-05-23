package kubernetes

type node struct {
	id      string
	address string
	port    int
}

func (n *node) Id() string {
	return n.id
}

func (n *node) Address() string {
	return n.address
}

func (n *node) Port() int {
	return n.port
}

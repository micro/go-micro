package registry

type Node interface {
	Id() string
	Address() string
	Port() int
}

func NewNode(id, address string, port int) Node {
	return DefaultRegistry.NewNode(id, address, port)
}

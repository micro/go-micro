package registry

type Service interface {
	Name() string
	Nodes() []Node
}

func NewService(name string, nodes ...Node) Service {
	return DefaultRegistry.NewService(name, nodes...)
}

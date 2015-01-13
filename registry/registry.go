package registry

type Registry interface {
	Register(Service) error
	Deregister(Service) error
	GetService(string) (Service, error)
	NewService(string, ...Node) Service
	NewNode(string, string, int) Node
}

var (
	DefaultRegistry = NewConsulRegistry()
)

func Register(s Service) error {
	return DefaultRegistry.Register(s)
}

func Deregister(s Service) error {
	return DefaultRegistry.Deregister(s)
}

func GetService(name string) (Service, error) {
	return DefaultRegistry.GetService(name)
}

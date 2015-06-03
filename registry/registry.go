package registry

type Registry interface {
	Register(*Service) error
	Deregister(*Service) error
	GetService(string) (*Service, error)
	ListServices() ([]*Service, error)
	Watch() (Watcher, error)
}

type Watcher interface {
	Stop()
}

type options struct{}

type Option func(*options)

var (
	DefaultRegistry = newConsulRegistry([]string{})
)

func NewRegistry(addrs []string, opt ...Option) Registry {
	return newConsulRegistry(addrs, opt...)
}

func Register(s *Service) error {
	return DefaultRegistry.Register(s)
}

func Deregister(s *Service) error {
	return DefaultRegistry.Deregister(s)
}

func GetService(name string) (*Service, error) {
	return DefaultRegistry.GetService(name)
}

func ListServices() ([]*Service, error) {
	return DefaultRegistry.ListServices()
}

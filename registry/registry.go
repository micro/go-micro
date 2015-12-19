package registry

type Registry interface {
	Register(*Service) error
	Deregister(*Service) error
	GetService(string) ([]*Service, error)
	ListServices() ([]*Service, error)
	Watch() (Watcher, error)
	String() string
}

type Option func(*Options)

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

func GetService(name string) ([]*Service, error) {
	return DefaultRegistry.GetService(name)
}

func ListServices() ([]*Service, error) {
	return DefaultRegistry.ListServices()
}

func Watch() (Watcher, error) {
	return DefaultRegistry.Watch()
}

func String() string {
	return DefaultRegistry.String()
}

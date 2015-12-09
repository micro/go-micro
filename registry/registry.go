package registry

import (
	"errors"
)

type Registry interface {
	Register(*Service) error
	Deregister(*Service) error
	GetService(string) ([]*Service, error)
	ListServices() ([]*Service, error)
	Watch() (Watcher, error)
}

type options struct{}

type Option func(*options)

var (
	DefaultRegistry = newConsulRegistry([]string{})

	ErrNotFound      = errors.New("not found")
	ErrNoneAvailable = errors.New("none available")
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

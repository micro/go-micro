// Package registry is an interface for service discovery
package registry

import (
	"errors"
)

// The registry provides an interface for service discovery
// and an abstraction over varying implementations
// {consul, etcd, zookeeper, ...}
type Registry interface {
	Register(*Service, ...RegisterOption) error
	Deregister(*Service) error
	GetService(string) ([]*Service, error)
	ListServices() ([]*Service, error)
	Watch(...WatchOption) (Watcher, error)
	String() string
	Options() Options
}

type Option func(*Options)

type RegisterOption func(*RegisterOptions)

type WatchOption func(*WatchOptions)

var (
	DefaultRegistry = newConsulRegistry()

	ErrNotFound = errors.New("not found")
)

func NewRegistry(opts ...Option) Registry {
	return newConsulRegistry(opts...)
}

// Package noop is a registry which does nothing
package noop

import (
	"errors"

	"github.com/micro/go-micro/v3/registry"
)

type noopRegistry struct{}

func (n *noopRegistry) Init(o ...registry.Option) error {
	return nil
}

func (n *noopRegistry) Options() registry.Options {
	return registry.Options{}
}

func (n *noopRegistry) Register(*registry.Service, ...registry.RegisterOption) error {
	return nil
}

func (n *noopRegistry) Deregister(*registry.Service, ...registry.DeregisterOption) error {
	return nil
}

func (n *noopRegistry) GetService(s string, o ...registry.GetOption) ([]*registry.Service, error) {
	return []*registry.Service{}, nil
}

func (n *noopRegistry) ListServices(...registry.ListOption) ([]*registry.Service, error) {
	return []*registry.Service{}, nil
}
func (n *noopRegistry) Watch(...registry.WatchOption) (registry.Watcher, error) {
	return nil, errors.New("not implemented")
}

func (n *noopRegistry) String() string {
	return "noop"
}

// NewRegistry returns a new noop registry
func NewRegistry(opts ...registry.Option) registry.Registry {
	return new(noopRegistry)
}

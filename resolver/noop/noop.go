// Package noop is a noop resolver
package noop

import (
	"github.com/micro/go-micro/v3/resolver"
)

type Resolver struct{}

// Resolve returns the list of nodes
func (r *Resolver) Resolve(name string) ([]*resolver.Record, error) {
	return []*resolver.Record{}, nil
}

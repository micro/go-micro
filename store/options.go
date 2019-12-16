package store

import (
	"github.com/micro/go-micro/config/options"
)

type Options struct {
	// nodes to connect to
	Nodes []string
	// Namespace of the store
	Namespace string
	// Prefix of the keys used
	Prefix string
}

// Nodes is a list of nodes used to back the store
func Nodes(a ...string) options.Option {
	return options.WithValue("store.nodes", a)
}

// Prefix sets a prefix to any key ids used
func Prefix(p string) options.Option {
	return options.WithValue("store.prefix", p)
}

// Namespace offers a way to have multiple isolated
// stores in the same backend, if supported.
func Namespace(n string) options.Option {
	return options.WithValue("store.namespace", n)
}

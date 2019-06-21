package store

import (
	"github.com/micro/go-micro/config/options"
)

// Set the nodes used to back the store
func Nodes(a ...string) options.Option {
	return options.WithValue("store.nodes", a)
}

// Prefix sets a prefix to any key ids used
func Prefix(p string) options.Option {
	return options.WithValue("store.prefix", p)
}

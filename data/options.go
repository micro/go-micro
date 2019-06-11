package data

import (
	"github.com/micro/go-micro/options"
)

// Set the nodes used to back the data
func Nodes(a ...string) options.Option {
	return options.WithValue("data.nodes", a)
}

// Prefix sets a prefix to any key ids used
func Prefix(p string) options.Option {
	return options.WithValue("data.prefix", p)
}

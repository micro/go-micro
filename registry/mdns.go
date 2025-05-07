//go:build !nats
// +build !nats

package registry

var (
	DefaultRegistry = NewRegistry()
)

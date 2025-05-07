//go:build nats
// +build nats

package cmd

import "go-micro.dev/v5/registry"

func init() {
	DefaultRegistries["nats"] = registry.NewRegistry
}

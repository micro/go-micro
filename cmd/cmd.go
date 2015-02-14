package cmd

import (
	"flag"

	"github.com/asim/go-micro/registry"
	"github.com/asim/go-micro/server"
	"github.com/asim/go-micro/store"
)

var (
	flagBindAddress string
	flagRegistry    string
	flagStore       string
)

func init() {
	flag.StringVar(&flagBindAddress, "bind_address", ":0", "Bind address for the server. 127.0.0.1:8080")
	flag.StringVar(&flagRegistry, "registry", "consul", "Registry for discovery. kubernetes, consul, etc")
	flag.StringVar(&flagStore, "store", "consul", "Store used as a basic key/value store using consul, memcached, etc")
}

func Init() {
	flag.Parse()

	server.Address = flagBindAddress

	switch flagRegistry {
	case "kubernetes":
		registry.DefaultRegistry = registry.NewKubernetesRegistry()
	}

	switch flagStore {
	case "memcached":
		store.DefaultStore = store.NewMemcacheStore()
	}
}

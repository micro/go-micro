// +build !minimal,!custom

package profile

import (
	"fmt"

	"github.com/micro/go-micro/v2/broker"
	eats "github.com/micro/go-micro/v2/broker/eats"
	_ "github.com/micro/go-micro/v2/broker/memory"
	_ "github.com/micro/go-micro/v2/broker/nats"
	_ "github.com/micro/go-micro/v2/broker/service"
	"github.com/micro/go-micro/v2/registry"
	_ "github.com/micro/go-micro/v2/registry/etcd"
	_ "github.com/micro/go-micro/v2/registry/kubernetes"
	mdns "github.com/micro/go-micro/v2/registry/mdns"
	_ "github.com/micro/go-micro/v2/registry/memory"
	_ "github.com/micro/go-micro/v2/registry/service"
)

func init() {
	fmt.Printf("init default profile\n")
	registry.DefaultRegistry = mdns.NewRegistry()
	broker.DefaultBroker = eats.NewBroker()
}

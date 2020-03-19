// +build minimal,!custom

package profile

import (
	"fmt"

	"github.com/micro/go-micro/v2/broker"
	bmemory "github.com/micro/go-micro/v2/broker/memory"
	"github.com/micro/go-micro/v2/registry"
	rmemory "github.com/micro/go-micro/v2/registry/memory"
)

func init() {
	fmt.Printf("init minimal profile\n")
	registry.DefaultRegistry = rmemory.NewRegistry()
	broker.DefaultBroker = bmemory.NewBroker()
}

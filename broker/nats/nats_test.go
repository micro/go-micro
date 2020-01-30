package nats

import (
	"fmt"
	"testing"

	"github.com/micro/go-micro/v2/broker"
	nats "github.com/nats-io/nats.go"
)

var addrTestCases = []struct {
	name        string
	description string
	addrs       map[string]string // expected address : set address
}{
	{
		"brokerOpts",
		"set broker addresses through a broker.Option in constructor",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"brokerInit",
		"set broker addresses through a broker.Option in broker.Init()",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"natsOpts",
		"set broker addresses through the nats.Option in constructor",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"default",
		"check if default Address is set correctly",
		map[string]string{
			"nats://127.0.0.1:4222": "",
		},
	},
}

// TestInitAddrs tests issue #100. Ensures that if the addrs is set by an option in init it will be used.
func TestInitAddrs(t *testing.T) {

	for _, tc := range addrTestCases {
		t.Run(fmt.Sprintf("%s: %s", tc.name, tc.description), func(t *testing.T) {

			var br broker.Broker
			var addrs []string

			for _, addr := range tc.addrs {
				addrs = append(addrs, addr)
			}

			switch tc.name {
			case "brokerOpts":
				// we know that there are just two addrs in the dict
				br = NewBroker(broker.Addrs(addrs[0], addrs[1]))
				br.Init()
			case "brokerInit":
				br = NewBroker()
				// we know that there are just two addrs in the dict
				br.Init(broker.Addrs(addrs[0], addrs[1]))
			case "natsOpts":
				nopts := nats.GetDefaultOptions()
				nopts.Servers = addrs
				br = NewBroker(Options(nopts))
				br.Init()
			case "default":
				br = NewBroker()
				br.Init()
			}

			natsBroker, ok := br.(*natsBroker)
			if !ok {
				t.Fatal("Expected broker to be of types *natsBroker")
			}
			// check if the same amount of addrs we set has actually been set, default
			// have only 1 address nats://127.0.0.1:4222 (current nats code) or
			// nats://localhost:4222 (older code version)
			if len(natsBroker.addrs) != len(tc.addrs) && tc.name != "default" {
				t.Errorf("Expected Addr count = %d, Actual Addr count = %d",
					len(natsBroker.addrs), len(tc.addrs))
			}

			for _, addr := range natsBroker.addrs {
				_, ok := tc.addrs[addr]
				if !ok {
					t.Errorf("Expected '%s' has not been set", addr)
				}
			}
		})
	}
}

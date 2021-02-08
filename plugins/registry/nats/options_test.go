package nats

import (
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/asim/go-micro/v3/registry"
	"github.com/nats-io/nats.go"
)

var addrTestCases = []struct {
	name        string
	description string
	addrs       map[string]string // expected address : set address
}{
	{
		"registryOption",
		"set registry addresses through a registry.Option in constructor",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"natsOption",
		"set registry addresses through the nats.Option in constructor",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"default",
		"check if default Address is set correctly",
		map[string]string{
			"nats://localhost:4222": ""},
	},
}

func TestInitAddrs(t *testing.T) {

	for _, tc := range addrTestCases {
		t.Run(fmt.Sprintf("%s: %s", tc.name, tc.description), func(t *testing.T) {

			var reg registry.Registry
			var addrs []string

			for _, addr := range tc.addrs {
				addrs = append(addrs, addr)
			}

			switch tc.name {
			case "registryOption":
				// we know that there are just two addrs in the dict
				reg = NewRegistry(registry.Addrs(addrs[0], addrs[1]))
			case "natsOption":
				nopts := nats.GetDefaultOptions()
				nopts.Servers = addrs
				reg = NewRegistry(Options(nopts))
			case "default":
				reg = NewRegistry()
			}

			// if err := reg.Register(dummyService); err != nil {
			// 	t.Fatal(err)
			// }

			natsRegistry, ok := reg.(*natsRegistry)
			if !ok {
				t.Fatal("Expected registry to be of types *natsRegistry")
			}
			// check if the same amount of addrs we set has actually been set
			if len(natsRegistry.addrs) != len(tc.addrs) {
				t.Errorf("Expected Addr count = %d, Actual Addr count = %d",
					len(natsRegistry.addrs), len(tc.addrs))
			}

			for _, addr := range natsRegistry.addrs {
				_, ok := tc.addrs[addr]
				if !ok {
					t.Errorf("Expected '%s' has not been set", addr)
				}
			}
		})

	}
}

func TestWatchQueryTopic(t *testing.T) {

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		log.Println("NATS_URL is undefined - skipping tests")
		return
	}

	watchTopic := "custom.test.watch"
	queryTopic := "custom.test.query"
	wt := WatchTopic(watchTopic)
	qt := QueryTopic(queryTopic)

	// connect to NATS and subscribe to the Watch & Query topics where the
	// registry will publish a msg
	nopts := nats.GetDefaultOptions()
	nopts.Servers = setAddrs([]string{natsURL})
	conn, err := nopts.Connect()
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	okCh := make(chan struct{})

	// Wait until we have received something on both topics
	go func() {
		wg.Wait()
		close(okCh)
	}()

	// handler just calls wg.Done()
	rcvdHdlr := func(m *nats.Msg) {
		wg.Done()
	}

	_, err = conn.Subscribe(queryTopic, rcvdHdlr)
	if err != nil {
		t.Fatal(err)
	}

	_, err = conn.Subscribe(watchTopic, rcvdHdlr)
	if err != nil {
		t.Fatal(err)
	}

	dummyService := &registry.Service{
		Name:    "TestInitAddr",
		Version: "1.0.0",
	}

	reg := NewRegistry(qt, wt, registry.Addrs(natsURL))

	// trigger registry to send out message on watchTopic
	if err := reg.Register(dummyService); err != nil {
		t.Fatal(err)
	}

	// trigger registry to send out message on queryTopic
	if _, err := reg.ListServices(); err != nil {
		t.Fatal(err)
	}

	// make sure that we received something on tc.topic
	select {
	case <-okCh:
		// fine - we received on both topics a message from the registry
	case <-time.After(time.Millisecond * 200):
		t.Fatal("timeout - no data received on watch topic")
	}
}

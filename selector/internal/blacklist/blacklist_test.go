package blacklist

import (
	"errors"
	"testing"
	"time"

	"github.com/micro/go-micro/registry"
)

func TestBlackList(t *testing.T) {
	bl := &BlackList{
		ttl:  1,
		bl:   make(map[string]blackListNode),
		exit: make(chan bool),
	}

	go bl.run()
	defer bl.Close()

	services := []*registry.Service{
		&registry.Service{
			Name: "foo",
			Nodes: []*registry.Node{
				&registry.Node{
					Id:      "foo-1",
					Address: "localhost",
					Port:    10001,
				},
				&registry.Node{
					Id:      "foo-2",
					Address: "localhost",
					Port:    10002,
				},
				&registry.Node{
					Id:      "foo-3",
					Address: "localhost",
					Port:    10002,
				},
			},
		},
	}

	// check nothing is filtered on clean run
	filterTest := func() {
		for i := 0; i < 3; i++ {
			srvs, err := bl.Filter(services)
			if err != nil {
				t.Fatal(err)
			}

			if len(srvs) != len(services) {
				t.Fatal("nodes were filtered when they shouldn't be")
			}

			for _, node := range srvs[0].Nodes {
				var seen bool
				for _, n := range srvs[0].Nodes {
					if n.Id == node.Id {
						seen = true
						break
					}
				}
				if !seen {
					t.Fatalf("Missing node %s", node.Id)
				}
			}
		}
	}

	// run filter test
	filterTest()

	blacklistTest := func() {
		// test blacklisting
		// mark until failure
		for i := 0; i < count+1; i++ {
			for _, node := range services[0].Nodes {
				bl.Mark("foo", node, errors.New("blacklist"))
			}
		}

		filtered, err := bl.Filter(services)
		if err != nil {
			t.Fatal(err)
		}

		if len(filtered) > 0 {
			t.Fatalf("Expected zero nodes got %+v", filtered)
		}
	}

	// sleep the ttl duration
	time.Sleep(time.Second * time.Duration(bl.ttl) * 2)

	// now run filterTest again
	filterTest()

	// run the blacklist test
	blacklistTest()

	// reset
	bl.Reset("foo")

	// check again
	filterTest()
}

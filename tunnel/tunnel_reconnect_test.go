// +build !race

package tunnel

import (
	"sync"
	"testing"
	"time"
)

func TestReconnectTunnel(t *testing.T) {
	// create a new tunnel client
	tunA := NewTunnel(
		Address("127.0.0.1:9096"),
		Nodes("127.0.0.1:9097"),
	)

	// create a new tunnel server
	tunB := NewTunnel(
		Address("127.0.0.1:9097"),
	)

	// start tunnel
	err := tunB.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer tunB.Close()

	// we manually override the tunnel.ReconnectTime value here
	// this is so that we make the reconnects faster than the default 5s
	ReconnectTime = 200 * time.Millisecond

	// start tunnel
	err = tunA.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer tunA.Close()

	wait := make(chan bool)

	var wg sync.WaitGroup

	wg.Add(1)
	// start tunnel listener
	go testBrokenTunAccept(t, tunB, wait, &wg)

	wg.Add(1)
	// start tunnel sender
	go testBrokenTunSend(t, tunA, wait, &wg)

	// wait until done
	wg.Wait()
}

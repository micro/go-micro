package net

import (
	"net"
	"testing"
)

func TestListen(t *testing.T) {
	fn := func(addr string) (net.Listener, error) {
		return net.Listen("tcp", addr)
	}

	// try to create a number of listeners
	for i := 0; i < 10; i++ {
		l, err := Listen("localhost:10000-11000", fn)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
	}
}

package net

import (
	"net"
	"os"
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

	// TODO nats case test
	// natsAddr := "_INBOX.bID2CMRvlNp0vt4tgNBHWf"
	// Expect addr DO NOT has extra ":" at the end!

}

// TestProxyEnv checks whether we have proxy/network settings in env
func TestProxyEnv(t *testing.T) {
	service := "foo"
	address := []string{"bar"}

	s, a, ok := Proxy(service, address)
	if ok {
		t.Fatal("Should not have proxy", s, a, ok)
	}

	test := func(key, val, expectSrv, expectAddr string) {
		// set env
		os.Setenv(key, val)

		s, a, ok := Proxy(service, address)
		if !ok {
			t.Fatal("Expected proxy")
		}
		if len(expectSrv) > 0 && s != expectSrv {
			t.Fatal("Expected proxy service", expectSrv, "got", s)
		}
		if len(expectAddr) > 0 {
			if len(a) == 0 || a[0] != expectAddr {
				t.Fatal("Expected proxy address", expectAddr, "got", a)
			}
		}

		os.Unsetenv(key)
	}

	test("MICRO_PROXY", "service", "go.micro.proxy", "")
	test("MICRO_NETWORK", "service", "go.micro.network", "")
	test("MICRO_NETWORK_ADDRESS", "10.0.0.1:8081", "", "10.0.0.1:8081")
}

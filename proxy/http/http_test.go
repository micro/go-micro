package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/micro/go-micro/v3/client"
	cmucp "github.com/micro/go-micro/v3/client/mucp"
	"github.com/micro/go-micro/v3/registry/memory"
	"github.com/micro/go-micro/v3/router"
	"github.com/micro/go-micro/v3/router/registry"
	"github.com/micro/go-micro/v3/server"
	"github.com/micro/go-micro/v3/server/mucp"
)

type testHandler struct{}

func (t *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"hello": "world"}`))
}

func TestHTTPProxy(t *testing.T) {
	c, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	addr := c.Addr().String()

	url := fmt.Sprintf("http://%s", addr)

	testCases := []struct {
		// http endpoint to call e.g /foo/bar
		httpEp string
		// rpc endpoint called e.g Foo.Bar
		rpcEp string
		// should be an error
		err bool
	}{
		{"/", "Foo.Bar", false},
		{"/", "Foo.Baz", false},
		{"/helloworld", "Hello.World", true},
	}

	// handler
	http.Handle("/", new(testHandler))

	// new proxy
	p := NewSingleHostProxy(url)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reg := memory.NewRegistry()
	rtr := registry.NewRouter(
		router.Registry(reg),
	)

	// new micro service
	service := mucp.NewServer(
		server.Context(ctx),
		server.Name("foobar"),
		server.Registry(reg),
		server.WithRouter(p),
	)

	service.Start()
	defer service.Stop()

	// run service
	// server
	go http.Serve(c, nil)

	cl := cmucp.NewClient(
		client.Router(rtr),
	)

	for _, test := range testCases {
		req := cl.NewRequest("foobar", test.rpcEp, map[string]string{"foo": "bar"}, client.WithContentType("application/json"))
		var rsp map[string]string
		err := cl.Call(ctx, req, &rsp)
		if err != nil && test.err == false {
			t.Fatal(err)
		}
		if v := rsp["hello"]; v != "world" {
			t.Fatalf("Expected hello world got %s from %s", v, test.rpcEp)
		}
	}
}

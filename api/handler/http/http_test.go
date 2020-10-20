package http

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asim/go-micro/v3/api/handler"
	"github.com/asim/go-micro/v3/api/resolver"
	rpath "github.com/asim/go-micro/v3/api/resolver/path"
	"github.com/asim/go-micro/v3/api/router"
	regRouter "github.com/asim/go-micro/v3/api/router/registry"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/registry/memory"
)

func testHttp(t *testing.T, path, service, ns string) {
	r := memory.NewRegistry()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	s := &registry.Service{
		Name: service,
		Nodes: []*registry.Node{
			{
				Id:      service + "-1",
				Address: l.Addr().String(),
			},
		},
	}

	r.Register(s)
	defer r.Deregister(s)

	// setup the test handler
	m := http.NewServeMux()
	m.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`you got served`))
	})

	// start http test serve
	go http.Serve(l, m)

	// create new request and writer
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", path, nil)
	if err != nil {
		t.Fatal(err)
	}

	// initialise the handler
	rt := regRouter.NewRouter(
		router.WithHandler("http"),
		router.WithRegistry(r),
		router.WithResolver(rpath.NewResolver(
			resolver.WithServicePrefix(ns),
		)),
	)

	p := NewHandler(handler.WithRouter(rt))

	// execute the handler
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200 response got %d %s", w.Code, w.Body.String())
	}

	if w.Body.String() != "you got served" {
		t.Fatalf("Expected body: you got served. Got: %s", w.Body.String())
	}
}

func TestHttpHandler(t *testing.T) {
	testData := []struct {
		path      string
		service   string
		namespace string
	}{
		{
			"/test/foo",
			"go.micro.api.test",
			"go.micro.api",
		},
		{
			"/test/foo/baz",
			"go.micro.api.test",
			"go.micro.api",
		},
	}

	for _, d := range testData {
		t.Run(d.service, func(t *testing.T) {
			testHttp(t, d.path, d.service, d.namespace)
		})
	}
}

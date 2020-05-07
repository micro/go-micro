package router_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/api"
	"github.com/micro/go-micro/v2/api/handler"
	"github.com/micro/go-micro/v2/api/handler/rpc"
	"github.com/micro/go-micro/v2/api/router"
	rregistry "github.com/micro/go-micro/v2/api/router/registry"
	rstatic "github.com/micro/go-micro/v2/api/router/static"
	"github.com/micro/go-micro/v2/client"
	gcli "github.com/micro/go-micro/v2/client/grpc"
	rmemory "github.com/micro/go-micro/v2/registry/memory"
	"github.com/micro/go-micro/v2/server"
	gsrv "github.com/micro/go-micro/v2/server/grpc"
	pb "github.com/micro/go-micro/v2/server/grpc/proto"
)

// server is used to implement helloworld.GreeterServer.
type testServer struct {
	msgCount int
}

// TestHello implements helloworld.GreeterServer
func (s *testServer) Call(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Msg = "Hello " + req.Uuid
	return nil
}

// TestHello implements helloworld.GreeterServer
func (s *testServer) CallPcre(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Msg = "Hello " + req.Uuid
	return nil
}

// TestHello implements helloworld.GreeterServer
func (s *testServer) CallPcreInvalid(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Msg = "Hello " + req.Uuid
	return nil
}

func initial(t *testing.T) (server.Server, client.Client) {
	r := rmemory.NewRegistry()

	// create a new client
	s := gsrv.NewServer(
		server.Name("foo"),
		server.Registry(r),
	)

	// create a new server
	c := gcli.NewClient(
		client.Registry(r),
	)

	h := &testServer{}
	pb.RegisterTestHandler(s, h)

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	return s, c
}

func check(t *testing.T, addr string, path string, expected string) {
	req, err := http.NewRequest("POST", fmt.Sprintf(path, addr), nil)
	if err != nil {
		t.Fatalf("Failed to created http.Request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	rsp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("Failed to created http.Request: %v", err)
	}
	defer rsp.Body.Close()

	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		t.Fatal(err)
	}

	jsonMsg := expected
	if string(buf) != jsonMsg {
		t.Fatalf("invalid message received, parsing error %s != %s", buf, jsonMsg)
	}
}

func TestRouterRegistryPcre(t *testing.T) {
	s, c := initial(t)
	defer s.Stop()

	router := rregistry.NewRouter(
		router.WithHandler(rpc.Handler),
		router.WithRegistry(s.Options().Registry),
	)
	hrpc := rpc.NewHandler(
		handler.WithClient(c),
		handler.WithRouter(router),
	)
	hsrv := &http.Server{
		Handler:        hrpc,
		Addr:           "127.0.0.1:6543",
		WriteTimeout:   15 * time.Second,
		ReadTimeout:    15 * time.Second,
		IdleTimeout:    20 * time.Second,
		MaxHeaderBytes: 1024 * 1024 * 1, // 1Mb
	}

	go func() {
		log.Println(hsrv.ListenAndServe())
	}()

	defer hsrv.Close()
	time.Sleep(1 * time.Second)
	check(t, hsrv.Addr, "http://%s/api/v0/test/call/TEST", `{"msg":"Hello TEST"}`)
}

func TestRouterStaticPcre(t *testing.T) {
	s, c := initial(t)
	defer s.Stop()

	router := rstatic.NewRouter(
		router.WithHandler(rpc.Handler),
		router.WithRegistry(s.Options().Registry),
	)

	err := router.Register(&api.Endpoint{
		Name:    "foo.Test.Call",
		Method:  []string{"POST"},
		Path:    []string{"^/api/v0/test/call/?$"},
		Handler: "rpc",
	})
	if err != nil {
		t.Fatal(err)
	}

	hrpc := rpc.NewHandler(
		handler.WithClient(c),
		handler.WithRouter(router),
	)
	hsrv := &http.Server{
		Handler:        hrpc,
		Addr:           "127.0.0.1:6543",
		WriteTimeout:   15 * time.Second,
		ReadTimeout:    15 * time.Second,
		IdleTimeout:    20 * time.Second,
		MaxHeaderBytes: 1024 * 1024 * 1, // 1Mb
	}

	go func() {
		log.Println(hsrv.ListenAndServe())
	}()
	defer hsrv.Close()

	time.Sleep(1 * time.Second)
	check(t, hsrv.Addr, "http://%s/api/v0/test/call", `{"msg":"Hello "}`)
}

func TestRouterStaticGpath(t *testing.T) {
	s, c := initial(t)
	defer s.Stop()

	router := rstatic.NewRouter(
		router.WithHandler(rpc.Handler),
		router.WithRegistry(s.Options().Registry),
	)

	err := router.Register(&api.Endpoint{
		Name:    "foo.Test.Call",
		Method:  []string{"POST"},
		Path:    []string{"/api/v0/test/call/{uuid}"},
		Handler: "rpc",
	})
	if err != nil {
		t.Fatal(err)
	}

	hrpc := rpc.NewHandler(
		handler.WithClient(c),
		handler.WithRouter(router),
	)
	hsrv := &http.Server{
		Handler:        hrpc,
		Addr:           "127.0.0.1:6543",
		WriteTimeout:   15 * time.Second,
		ReadTimeout:    15 * time.Second,
		IdleTimeout:    20 * time.Second,
		MaxHeaderBytes: 1024 * 1024 * 1, // 1Mb
	}

	go func() {
		log.Println(hsrv.ListenAndServe())
	}()
	defer hsrv.Close()

	time.Sleep(1 * time.Second)
	check(t, hsrv.Addr, "http://%s/api/v0/test/call/TEST", `{"msg":"Hello TEST"}`)
}

func TestRouterStaticPcreInvalid(t *testing.T) {
	var ep *api.Endpoint
	var err error

	s, c := initial(t)
	defer s.Stop()

	router := rstatic.NewRouter(
		router.WithHandler(rpc.Handler),
		router.WithRegistry(s.Options().Registry),
	)

	ep = &api.Endpoint{
		Name:    "foo.Test.Call",
		Method:  []string{"POST"},
		Path:    []string{"^/api/v0/test/call/?"},
		Handler: "rpc",
	}

	err = router.Register(ep)
	if err == nil {
		t.Fatalf("invalid endpoint %v", ep)
	}

	ep = &api.Endpoint{
		Name:    "foo.Test.Call",
		Method:  []string{"POST"},
		Path:    []string{"/api/v0/test/call/?$"},
		Handler: "rpc",
	}

	err = router.Register(ep)
	if err == nil {
		t.Fatalf("invalid endpoint %v", ep)
	}

	_ = c
}

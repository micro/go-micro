package api_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/api"
	ahandler "github.com/micro/go-micro/v2/api/handler"
	apirpc "github.com/micro/go-micro/v2/api/handler/rpc"
	"github.com/micro/go-micro/v2/api/router"
	rstatic "github.com/micro/go-micro/v2/api/router/static"
	bmemory "github.com/micro/go-micro/v2/broker/memory"
	"github.com/micro/go-micro/v2/client"
	gcli "github.com/micro/go-micro/v2/client/grpc"
	rmemory "github.com/micro/go-micro/v2/registry/memory"
	"github.com/micro/go-micro/v2/server"
	gsrv "github.com/micro/go-micro/v2/server/grpc"
	tgrpc "github.com/micro/go-micro/v2/transport/grpc"

	pb "github.com/micro/go-micro/v2/server/grpc/proto"
)

// server is used to implement helloworld.GreeterServer.
type testServer struct {
	msgCount int
}

// TestHello implements helloworld.GreeterServer
func (s *testServer) Call(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Msg = "Hello " + req.Name
	return nil
}

func TestApiAndGRPC(t *testing.T) {
	r := rmemory.NewRegistry()
	b := bmemory.NewBroker()
	tr := tgrpc.NewTransport()
	s := gsrv.NewServer(
		server.Broker(b),
		server.Name("foo"),
		server.Registry(r),
		server.Transport(tr),
	)
	c := gcli.NewClient(
		client.Registry(r),
		client.Broker(b),
		client.Transport(tr),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := micro.NewService(
		micro.Server(s),
		micro.Client(c),
		micro.Broker(b),
		micro.Registry(r),
		micro.Transport(tr),
		micro.Context(ctx))
	h := &testServer{}
	pb.RegisterTestHandler(s, h)

	go func() {
		if err := svc.Run(); err != nil {
			t.Fatalf("failed to start: %v", err)
		}
	}()
	time.Sleep(1 * time.Second)
	// check registration
	services, err := r.GetService("foo")
	if err != nil || len(services) == 0 {
		t.Fatalf("failed to get service: %v # %d", err, len(services))
	}

	router := rstatic.NewRouter(
		router.WithHandler(apirpc.Handler),
		router.WithRegistry(svc.Server().Options().Registry),
	)
	err = router.Register(&api.Endpoint{
		Name:    "foo.Test.Call",
		Method:  []string{"GET"},
		Path:    []string{"/api/v0/test/call/{name}"},
		Handler: "rpc",
	})
	if err != nil {
		t.Fatal(err)
	}

	hrpc := apirpc.NewHandler(
		ahandler.WithService(svc),
		ahandler.WithRouter(router),
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

	time.Sleep(1 * time.Second)
	rsp, err := http.Get(fmt.Sprintf("http://%s/api/v0/test/call/TEST", hsrv.Addr))
	if err != nil {
		t.Fatalf("Failed to created http.Request: %v", err)
	}
	defer rsp.Body.Close()
	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		t.Fatal(err)
	}

	jsonMsg := `{"msg":"Hello TEST"}`
	if string(buf) != jsonMsg {
		t.Fatalf("invalid message received, parsing error %s != %s", buf, jsonMsg)
	}
	select {
	case <-ctx.Done():
		return
	}

}

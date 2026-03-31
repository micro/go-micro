package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	micro "go-micro.dev/v5"
	"go-micro.dev/v5/client"
	grpcclient "go-micro.dev/v5/client/grpc"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/server"
	grpcserver "go-micro.dev/v5/server/grpc"
)

type SleepRequest struct {
	DelayMS int `json:"delay_ms"`
}

type SleepResponse struct {
	Message string `json:"message"`
}

type Sleeper struct {
	started chan struct{}
}

func (s *Sleeper) Sleep(ctx context.Context, req *SleepRequest, rsp *SleepResponse) error {
	select {
	case s.started <- struct{}{}:
	default:
	}

	timer := time.NewTimer(time.Duration(req.DelayMS) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	rsp.Message = fmt.Sprintf("slept for %dms", req.DelayMS)
	return nil
}

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	addr := listener.Addr().String()
	reg := registry.NewMemoryRegistry()
	handler := &Sleeper{started: make(chan struct{}, 1)}

	service := micro.New("grace-demo",
		micro.HandleSignal(false),
		micro.Registry(reg),
		micro.Server(grpcserver.NewServer(
			server.Registry(reg),
			server.Name("grace-demo"),
			server.Address(addr),
			grpcserver.Listener(listener),
			grpcserver.GracefulStopTimeout(3*time.Second),
		)),
		micro.Client(grpcclient.NewClient(
			client.Registry(reg),
			client.ContentType("application/grpc+json"),
			client.DialTimeout(200*time.Millisecond),
			client.RequestTimeout(5*time.Second),
		)),
	)

	if err := service.Handle(handler); err != nil {
		log.Fatal(err)
	}
	if err := service.Start(); err != nil {
		log.Fatal(err)
	}

	log.Printf("service started on %s", addr)

	longDone := make(chan error, 1)
	go func() {
		req := service.Client().NewRequest("grace-demo", "Sleeper.Sleep", &SleepRequest{DelayMS: 1500})
		rsp := &SleepResponse{}
		longDone <- service.Client().Call(context.Background(), req, rsp, client.WithAddress(addr))
		if rsp.Message != "" {
			log.Printf("long RPC completed: %s", rsp.Message)
		}
	}()

	<-handler.started
	log.Printf("long RPC is running; starting shutdown")

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- service.Stop()
	}()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		callCtx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		req := service.Client().NewRequest("grace-demo", "Sleeper.Sleep", &SleepRequest{DelayMS: 50})
		rsp := &SleepResponse{}
		err = service.Client().Call(callCtx, req, rsp, client.WithAddress(addr))
		cancel()
		if err != nil {
			log.Printf("new RPC rejected after shutdown began: %v", err)
			break
		}

		log.Printf("new RPC still accepted during shutdown: %s", rsp.Message)
		time.Sleep(10 * time.Millisecond)
	}

	if err := <-longDone; err != nil {
		log.Fatalf("long RPC failed: %v", err)
	}
	if err := <-stopDone; err != nil {
		log.Fatalf("service stop failed: %v", err)
	}

	log.Printf("done")
}

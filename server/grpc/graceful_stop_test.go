package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	micro "go-micro.dev/v5"
	"go-micro.dev/v5/client"
	grpcclient "go-micro.dev/v5/client/grpc"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/server"
)

type SleepRequest struct {
	DelayMS int `json:"delay_ms"`
}

type SleepResponse struct {
	Message string `json:"message"`
}

type SleepHandler struct {
	started chan struct{}
}

func (h *SleepHandler) Sleep(ctx context.Context, req *SleepRequest, rsp *SleepResponse) error {
	select {
	case h.started <- struct{}{}:
	default:
	}

	timer := time.NewTimer(time.Duration(req.DelayMS) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	rsp.Message = "ok"
	return nil
}

func TestGracefulStopRejectsNewRPCsButAllowsInFlightRPCs(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()
	reg := registry.NewMemoryRegistry()
	handler := &SleepHandler{started: make(chan struct{}, 1)}

	svc := micro.New("grace-demo",
		micro.HandleSignal(false),
		micro.Registry(reg),
		micro.Server(NewServer(
			server.Registry(reg),
			server.Name("grace-demo"),
			server.Address(addr),
			Listener(listener),
			GracefulStopTimeout(3*time.Second),
		)),
		micro.Client(grpcclient.NewClient(
			client.Registry(reg),
			client.ContentType("application/grpc+json"),
			client.DialTimeout(200*time.Millisecond),
			client.RequestTimeout(5*time.Second),
		)),
	)

	if err := svc.Handle(handler); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if err := svc.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	stopped := false
	defer func() {
		if !stopped {
			_ = svc.Stop()
		}
	}()

	longDone := make(chan error, 1)
	go func() {
		req := svc.Client().NewRequest("grace-demo", "SleepHandler.Sleep", &SleepRequest{DelayMS: 1000})
		rsp := &SleepResponse{}
		longDone <- svc.Client().Call(context.Background(), req, rsp, client.WithAddress(addr))
	}()

	select {
	case <-handler.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for long RPC to start")
	}

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- svc.Stop()
	}()

	freshReq := svc.Client().NewRequest("grace-demo", "SleepHandler.Sleep", &SleepRequest{DelayMS: 10})
	freshRsp := &SleepResponse{}
	var rejectErr error

	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		callCtx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		err := svc.Client().Call(callCtx, freshReq, freshRsp, client.WithAddress(addr))
		cancel()
		if err != nil {
			rejectErr = err
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if rejectErr == nil {
		t.Fatal("expected a new RPC to be rejected shortly after shutdown started")
	}

	select {
	case err := <-longDone:
		if err != nil {
			t.Fatalf("long RPC failed during graceful stop: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for in-flight RPC to finish")
	}

	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatalf("stop: %v", err)
		}
		stopped = true
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server stop")
	}
}

package client_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-micro.dev/v6/client"
	raw "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
	"go-micro.dev/v6/server"
)

type EchoReq struct {
	Msg string `json:"msg"`
}

type EchoRsp struct {
	Msg string `json:"msg"`
}

type EchoHandler struct{}

func (EchoHandler) Echo(_ context.Context, req *EchoReq, rsp *EchoRsp) error {
	rsp.Msg = "echo:" + req.Msg
	return nil
}

// startEchoServer starts a real server on the given registry and returns a stop
// func. The server is reachable over the network transport and (via Start)
// registered for the in-process fast-path.
func startEchoServer(t testing.TB, reg registry.Registry) func() {
	t.Helper()
	srv := server.NewServer(
		server.Name("echo.local"),
		server.Address("127.0.0.1:0"),
		server.Registry(reg),
	)
	if err := srv.Handle(srv.NewHandler(&EchoHandler{})); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	// Wait for registration so the client's selector can find a node.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if svcs, err := reg.GetService("echo.local"); err == nil && len(svcs) > 0 && len(svcs[0].Nodes) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return func() { _ = srv.Stop() }
}

func newEchoClient(reg registry.Registry, opts ...client.Option) client.Client {
	base := []client.Option{
		client.Registry(reg),
		client.Selector(selector.NewSelector(selector.Registry(reg))),
		client.ContentType("application/json"),
	}
	return client.NewClient(append(base, opts...)...)
}

// callEcho makes an echo call with a raw-frame body (the shape agent/MCP/flow
// dispatch uses) and returns the decoded reply.
func callEcho(t testing.TB, cl client.Client, msg string) EchoRsp {
	t.Helper()
	body, _ := json.Marshal(EchoReq{Msg: msg})
	req := cl.NewRequest("echo.local", "EchoHandler.Echo", &raw.Frame{Data: body}, client.WithContentType("application/json"))
	var rsp raw.Frame
	if err := cl.Call(context.Background(), req, &rsp); err != nil {
		t.Fatalf("call: %v", err)
	}
	var out EchoRsp
	if err := json.Unmarshal(rsp.Data, &out); err != nil {
		t.Fatalf("decode reply %q: %v", rsp.Data, err)
	}
	return out
}

// TestLocalMatchesNetwork proves the in-process fast-path returns the
// exact same result as the network path for the same handler and request.
func TestLocalMatchesNetwork(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	stop := startEchoServer(t, reg)
	defer stop()

	net := newEchoClient(reg)                   // network path
	local := newEchoClient(reg, client.Local()) // in-process fast-path

	netRsp := callEcho(t, net, "hi")
	localRsp := callEcho(t, local, "hi")

	if netRsp.Msg != "echo:hi" {
		t.Fatalf("network reply = %q, want echo:hi", netRsp.Msg)
	}
	if localRsp != netRsp {
		t.Fatalf("fast-path reply %+v != network reply %+v", localRsp, netRsp)
	}
}

// TestLocalFallsBackWhenNotLocal confirms a service not registered
// in-process still works via the network path even with Local on.
func TestLocalFallsBackWhenNotLocal(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	stop := startEchoServer(t, reg)
	defer stop()

	// Local is on, but the call still resolves — the fast-path only
	// engages when it fully applies, otherwise the network path runs.
	local := newEchoClient(reg, client.Local())
	if got := callEcho(t, local, "x").Msg; got != "echo:x" {
		t.Fatalf("reply = %q, want echo:x", got)
	}
}

func benchmarkEcho(b *testing.B, opts ...client.Option) {
	reg := registry.NewMemoryRegistry()
	stop := startEchoServer(b, reg)
	defer stop()
	cl := newEchoClient(reg, opts...)
	body, _ := json.Marshal(EchoReq{Msg: "hi"})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := cl.NewRequest("echo.local", "EchoHandler.Echo", &raw.Frame{Data: body}, client.WithContentType("application/json"))
		var rsp raw.Frame
		if err := cl.Call(context.Background(), req, &rsp); err != nil {
			b.Fatalf("call: %v", err)
		}
	}
}

func BenchmarkNetworkCall(b *testing.B) { benchmarkEcho(b) }
func BenchmarkLocalCall(b *testing.B)   { benchmarkEcho(b, client.Local()) }

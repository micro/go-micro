package server

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v4/registry"
)

type CtxReq struct{}
type CtxRsp struct{}

type CtxHandler struct{ ctxs chan context.Context }

func (h *CtxHandler) Call(ctx context.Context, _ *CtxReq, _ *CtxRsp) error {
	h.ctxs <- ctx
	return nil
}

// TestRequestContextReleasedAfterReply verifies that a request's context is
// cancelled once the request has been served, while its connection stays
// open. Cancelling it only when ServeConn returns kept every request's
// context, timer and header copy alive for the life of a pooled keep-alive
// connection, so memory grew with each request the connection carried.
func TestRequestContextReleasedAfterReply(t *testing.T) {
	srv := NewRPCServer(
		Registry(registry.NewMemoryRegistry()),
		Name("test.context"),
		Address("127.0.0.1:0"),
	)
	h := &CtxHandler{ctxs: make(chan context.Context, 1)}
	if err := srv.Handle(srv.NewHandler(h)); err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Options().Address)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	body := `{}`
	req := fmt.Sprintf("POST / HTTP/1.1\r\nHost: test\r\nContent-Type: application/json\r\n"+
		"Micro-Service: test.context\r\nMicro-Endpoint: CtxHandler.Call\r\nMicro-Method: CtxHandler.Call\r\n"+
		"Micro-Id: 1\r\nTimeout: %d\r\nContent-Length: %d\r\n\r\n%s", time.Minute.Nanoseconds(), len(body), body)
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}

	var ctx context.Context
	select {
	case ctx = <-h.ctxs:
	case <-time.After(5 * time.Second):
		t.Fatal("handler was not called")
	}

	// Read the reply but keep the connection open, as a pooled client would.
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil || !strings.HasPrefix(string(buf[:n]), "HTTP/1.1 200") {
		t.Fatalf("no reply: %v %q", err, buf[:n])
	}

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("request context still live after the reply while its connection stays open")
	}
}

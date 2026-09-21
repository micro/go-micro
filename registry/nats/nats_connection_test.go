package nats

import (
	"errors"
	"testing"
	"time"

	server "github.com/nats-io/nats-server/v2/server"
	gonats "github.com/nats-io/nats.go"
	"go-micro.dev/v6/registry"
)

func TestQueryConnectionState(t *testing.T) {
	srv, err := server.NewServer(&server.Options{Host: "127.0.0.1", Port: -1})
	if err != nil {
		t.Fatalf("new NATS server: %v", err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("NATS server did not become ready")
	}
	t.Cleanup(func() {
		srv.Shutdown()
		srv.WaitForShutdown()
	})

	r := NewNatsRegistry(
		registry.Addrs(srv.ClientURL()),
		registry.Timeout(25*time.Millisecond),
	).(*natsRegistry)

	services, err := r.ListServices()
	if err != nil {
		t.Fatalf("query reachable NATS server: %v", err)
	}
	if len(services) != 0 {
		t.Fatalf("expected no services, got %d", len(services))
	}

	srv.Shutdown()
	srv.WaitForShutdown()
	deadline := time.Now().Add(time.Second)
	for r.conn.IsConnected() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if r.conn.IsConnected() {
		t.Fatal("connection remained connected after NATS stopped")
	}

	_, err = r.ListServices()
	if !errors.Is(err, gonats.ErrDisconnected) || !errors.Is(err, registry.ErrUnavailable) {
		t.Fatalf("expected disconnected error, got %v", err)
	}
}

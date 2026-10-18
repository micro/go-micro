package api

import (
	"net"
	"testing"
	"time"
)

func TestGatewayReportsOccupiedAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if gateway, err := New(Options{Address: listener.Addr().String()}); err == nil {
		_ = gateway.Stop()
		t.Fatal("expected address-in-use error")
	}
}

func TestGatewayWaitReturnsAfterStop(t *testing.T) {
	gateway, err := New(Options{Address: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- gateway.Wait() }()
	if err := gateway.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after Stop")
	}
}

package mock

import (
	"testing"

	"github.com/micro/go-micro/v2/server"
)

func TestMockServer(t *testing.T) {
	srv := NewServer(
		server.Name("mock"),
		server.Version("latest"),
	)

	if srv.Options().Name != "mock" {
		t.Fatalf("Expected name mock, got %s", srv.Options().Name)
	}

	if srv.Options().Version != "latest" {
		t.Fatalf("Expected version latest, got %s", srv.Options().Version)
	}

	srv.Init(server.Version("test"))
	if srv.Options().Version != "test" {
		t.Fatalf("Expected version test, got %s", srv.Options().Version)
	}

	h := srv.NewHandler(func() string { return "foo" })
	if err := srv.Handle(h); err != nil {
		t.Fatal(err)
	}

	sub := srv.NewSubscriber("test", func() string { return "foo" })
	if err := srv.Subscribe(sub); err != nil {
		t.Fatal(err)
	}

	if sub.Topic() != "test" {
		t.Fatalf("Expected topic test got %s", sub.Topic())
	}

	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}

	if err := srv.Register(); err != nil {
		t.Fatal(err)
	}

	if err := srv.Deregister(); err != nil {
		t.Fatal(err)
	}

	if err := srv.Stop(); err != nil {
		t.Fatal(err)
	}
}

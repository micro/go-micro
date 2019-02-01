package gossip_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
	micro "github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/gossip"
	pb "github.com/micro/go-micro/registry/gossip/proto"
	"github.com/micro/go-micro/selector"
	"github.com/micro/go-micro/server"
)

var (
	r1 registry.Registry
	r2 registry.Registry
	mu sync.Mutex
)

func newConfig() *memberlist.Config {
	wc := memberlist.DefaultLANConfig()
	wc.DisableTcpPings = false
	wc.GossipVerifyIncoming = false
	wc.GossipVerifyOutgoing = false
	wc.EnableCompression = false
	wc.LogOutput = os.Stderr
	wc.ProtocolVersion = 4
	wc.Name = uuid.New().String()
	return wc
}

func newRegistries() {
	mu.Lock()
	defer mu.Unlock()

	if r1 != nil && r2 != nil {
		return
	}

	wc1 := newConfig()
	wc2 := newConfig()

	rops1 := []registry.Option{gossip.Config(wc1), gossip.Address("127.0.0.1:54321")}
	rops2 := []registry.Option{gossip.Config(wc2), gossip.Address("127.0.0.2:54321"), registry.Addrs("127.0.0.1:54321")}

	r1 = gossip.NewRegistry(rops1...) // first started without members
	r2 = gossip.NewRegistry(rops2...) // second started joining
}

func TestRegistryBroadcast(t *testing.T) {
	newRegistries()

	svc1 := &registry.Service{Name: "r1-svc", Version: "0.0.0.1"}
	svc2 := &registry.Service{Name: "r2-svc", Version: "0.0.0.2"}

	<-time.After(1 * time.Second)
	if err := r1.Register(svc1); err != nil {
		t.Fatal(err)
	}
	<-time.After(1 * time.Second)
	if err := r2.Register(svc2); err != nil {
		t.Fatal(err)
	}

	var found bool

	svcs, err := r1.ListServices()
	if err != nil {
		t.Fatal(err)
	}

	for _, svc := range svcs {
		if svc.Name == "r2-svc" {
			found = true
		}
	}
	if !found {
		t.Fatalf("r2-svc not found in r1, broadcast not work")
	}

	found = false
	svcs, err = r2.ListServices()
	if err != nil {
		t.Fatal(err)
	}

	for _, svc := range svcs {
		if svc.Name == "r1-svc" {
			found = true
		}
	}
	if !found {
		t.Fatalf("r1-svc not found in r2, broadcast not work")
	}

	if err := r1.Deregister(svc1); err != nil {
		t.Fatal(err)
	}
	if err := r2.Deregister(svc2); err != nil {
		t.Fatal(err)
	}

}

func TestServerRegistry(t *testing.T) {
	newRegistries()

	_, err := newServer("s1", r1, t)
	if err != nil {
		t.Fatal(err)
	}

	_, err = newServer("s2", r2, t)
	if err != nil {
		t.Fatal(err)
	}

	svcs, err := r1.ListServices()
	if err != nil {
		t.Fatal(err)
	}
	if len(svcs) < 1 {
		t.Fatalf("r1 svcs unknown %#+v\n", svcs)
	}

	found := false
	for _, svc := range svcs {
		if svc.Name == "s2" {
			found = true
		}
	}
	if !found {
		t.Fatalf("r1 does not have s2, broadcast not work")
	}

	found = false
	svcs, err = r2.ListServices()
	if err != nil {
		t.Fatal(err)
	}

	for _, svc := range svcs {
		if svc.Name == "s1" {
			found = true
		}
	}

	if !found {
		t.Fatalf("r2 does not have s1, broadcast not work")
	}

}

type testServer struct{}

func (*testServer) Test(ctx context.Context, req *pb.Update, rsp *pb.Update) error {
	return nil
}

func newServer(n string, r registry.Registry, t *testing.T) (micro.Service, error) {
	h := &testServer{}

	var wg sync.WaitGroup

	wg.Add(1)
	sopts := []server.Option{
		server.Name(n),
		server.Registry(r),
	}

	copts := []client.Option{
		client.Selector(selector.NewSelector(selector.Registry(r))),
		client.Registry(r),
	}

	srv := micro.NewService(
		micro.Server(server.NewServer(sopts...)),
		micro.Client(client.NewClient(copts...)),
		micro.AfterStart(func() error {
			wg.Done()
			return nil
		}),
	)

	srv.Server().NewHandler(h)

	go func() {
		t.Fatal(srv.Run())
	}()
	wg.Wait()
	return srv, nil
}

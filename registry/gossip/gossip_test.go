package gossip

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
	"github.com/micro/go-micro/registry"
)

func newMemberlistConfig() *memberlist.Config {
	mc := memberlist.DefaultLANConfig()
	mc.DisableTcpPings = false
	mc.GossipVerifyIncoming = false
	mc.GossipVerifyOutgoing = false
	mc.EnableCompression = false
	mc.PushPullInterval = 3 * time.Second
	mc.LogOutput = os.Stderr
	mc.ProtocolVersion = 4
	mc.Name = uuid.New().String()
	return mc
}

func newRegistry(opts ...registry.Option) registry.Registry {
	options := []registry.Option{
		ConnectRetry(true),
		ConnectTimeout(60 * time.Second),
	}

	options = append(options, opts...)
	r := NewRegistry(options...)
	return r
}

func TestRegistryBroadcast(t *testing.T) {
	mc1 := newMemberlistConfig()
	r1 := newRegistry(Config(mc1), Address("127.0.0.1:54321"))

	mc2 := newMemberlistConfig()
	r2 := newRegistry(Config(mc2), Address("127.0.0.2:54321"), registry.Addrs("127.0.0.1:54321"))

	defer r1.(*gossipRegistry).Stop()
	defer r2.(*gossipRegistry).Stop()

	svc1 := &registry.Service{Name: "r1-svc", Version: "0.0.0.1"}
	svc2 := &registry.Service{Name: "r2-svc", Version: "0.0.0.2"}

	t.Logf("register service svc1 on r1\n")
	if err := r1.Register(svc1, registry.RegisterTTL(10*time.Second)); err != nil {
		t.Fatal(err)
	}
	t.Logf("register service svc2 on r2\n")
	if err := r2.Register(svc2, registry.RegisterTTL(10*time.Second)); err != nil {
		t.Fatal(err)
	}

	var found bool
	t.Logf("list services on r1\n")
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
	} else {
		t.Logf("r2-svc found in r1, all ok")
	}

	found = false
	t.Logf("list services on r2\n")
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
	} else {
		t.Logf("r1-svc found in r1, all ok")
	}

	t.Logf("deregister service svc1 on r1\n")
	if err := r1.Deregister(svc1); err != nil {
		t.Fatal(err)
	}
	t.Logf("deregister service svc1 on r2\n")
	if err := r2.Deregister(svc2); err != nil {
		t.Fatal(err)
	}

}
func TestRegistryRetry(t *testing.T) {
	mc1 := newMemberlistConfig()
	r1 := newRegistry(Config(mc1), Address("127.0.0.1:54321"))

	mc2 := newMemberlistConfig()
	r2 := newRegistry(Config(mc2), Address("127.0.0.2:54321"), registry.Addrs("127.0.0.1:54321"))

	defer r1.(*gossipRegistry).Stop()
	defer r2.(*gossipRegistry).Stop()

	svc1 := &registry.Service{Name: "r1-svc", Version: "0.0.0.1"}
	svc2 := &registry.Service{Name: "r2-svc", Version: "0.0.0.2"}

	var mu sync.Mutex
	ch := make(chan struct{})
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				if r1 != nil {
					r1.Register(svc1, registry.RegisterTTL(2*time.Second))
				}
				if r2 != nil {
					r2.Register(svc2, registry.RegisterTTL(2*time.Second))
				}
				if ch != nil {
					close(ch)
					ch = nil
				}
				mu.Unlock()
			}
		}
	}()

	<-ch
	var found bool

	svcs, err := r2.ListServices()
	if err != nil {
		t.Fatal(err)
	}

	for _, svc := range svcs {
		if svc.Name == "r1-svc" {
			found = true
		}
	}
	if !found {
		t.Fatalf("r1-svc not found in r2, broadcast not work, retry cant test")
	}

	t.Logf("stop r1\n")
	if err = r1.(*gossipRegistry).Stop(); err != nil {
		t.Fatalf("cant stop r1 registry %v", err)
	}

	mu.Lock()
	r1 = nil
	mu.Unlock()

	<-time.After(3 * time.Second)

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
	if found {
		t.Fatalf("r1-svc found in r2, something wrong")
	}

	t.Logf("start r1\n")

	r1 = newRegistry(Config(mc1), Address("127.0.0.1:54321"))
	<-time.After(2 * time.Second)

	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		t.Logf("skip next test part, becasue it not works in travis")
		t.Skip()
		return
		<-time.After(5 * time.Second)
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
		t.Fatalf("r1-svc not found in r2, connect retry not works")
	}

	if err := r1.Deregister(svc1); err != nil {
		t.Fatal(err)
	}
	if err := r2.Deregister(svc2); err != nil {
		t.Fatal(err)
	}

	r1.(*gossipRegistry).Stop()
	r2.(*gossipRegistry).Stop()
}

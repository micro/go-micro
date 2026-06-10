package etcd

import (
	"testing"
	"time"

	"go-micro.dev/v5/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func newTestRegistry() *etcdRegistry {
	e := &etcdRegistry{
		register:      map[string]uint64{},
		leases:        map[string]clientv3.LeaseID{},
		keepaliveChs:  map[string]<-chan *clientv3.LeaseKeepAliveResponse{},
		keepaliveStop: map[string]chan bool{},
	}
	e.options.Logger = logger.DefaultLogger
	return e
}

func seedLease(e *etcdRegistry, key string, ch <-chan *clientv3.LeaseKeepAliveResponse, stop chan bool) {
	e.Lock()
	e.keepaliveChs[key] = ch
	e.keepaliveStop[key] = stop
	e.leases[key] = clientv3.LeaseID(123)
	e.register[key] = 42
	e.Unlock()
}

func leaseCached(e *etcdRegistry, key string) bool {
	e.RLock()
	defer e.RUnlock()
	_, ok := e.leases[key]
	return ok
}

// A keepalive response with a non-positive TTL means the lease has
// expired server-side; the loop must drop the cached lease so the next
// Register re-registers instead of skipping on the "unchanged" check.
func TestKeepAliveLoopTTLExpiredDropsLease(t *testing.T) {
	e := newTestRegistry()
	key := "svc1node1"
	ch := make(chan *clientv3.LeaseKeepAliveResponse, 1)
	stop := make(chan bool, 1)
	seedLease(e, key, ch, stop)

	done := make(chan struct{})
	go func() { e.keepAliveLoop(key, ch, stop); close(done) }()

	ch <- &clientv3.LeaseKeepAliveResponse{ID: 123, TTL: 0}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("keepAliveLoop did not return after a TTL<=0 response")
	}
	if leaseCached(e, key) {
		t.Error("lease should be dropped after it expired (TTL<=0)")
	}
}

// A closed keepalive channel also drops the cached lease.
func TestKeepAliveLoopChannelClosedDropsLease(t *testing.T) {
	e := newTestRegistry()
	key := "svc1node1"
	ch := make(chan *clientv3.LeaseKeepAliveResponse)
	stop := make(chan bool, 1)
	seedLease(e, key, ch, stop)

	done := make(chan struct{})
	go func() { e.keepAliveLoop(key, ch, stop); close(done) }()

	close(ch)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("keepAliveLoop did not return after the channel closed")
	}
	if leaseCached(e, key) {
		t.Error("lease should be dropped after the channel closed")
	}
}

// A healthy response must NOT drop the lease, and stopping the loop (as
// on Deregister) leaves the cache for stopKeepAlive to clean up.
func TestKeepAliveLoopHealthyKeepsLease(t *testing.T) {
	e := newTestRegistry()
	key := "svc1node1"
	ch := make(chan *clientv3.LeaseKeepAliveResponse, 1)
	stop := make(chan bool, 1)
	seedLease(e, key, ch, stop)

	done := make(chan struct{})
	go func() { e.keepAliveLoop(key, ch, stop); close(done) }()

	ch <- &clientv3.LeaseKeepAliveResponse{ID: 123, TTL: 30}
	time.Sleep(50 * time.Millisecond)
	if !leaseCached(e, key) {
		t.Error("a healthy keepalive must not drop the lease")
	}

	stop <- true
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("keepAliveLoop did not return after stop")
	}
	if !leaseCached(e, key) {
		t.Error("stopping the loop should not drop the lease; stopKeepAlive does that")
	}
}

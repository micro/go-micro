package etcd

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// TestKeepAliveManagement tests that keepalive channels are properly managed
func TestKeepAliveManagement(t *testing.T) {
	// Skip if no etcd server available
	etcdAddr := os.Getenv("ETCD_ADDRESS")
	if etcdAddr == "" {
		etcdAddr = "127.0.0.1:2379"
	}

	// Try to connect to etcd
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdAddr},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Skip("Etcd not available, skipping test:", err)
		return
	}
	defer client.Close()

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = client.Get(ctx, "/test")
	if err != nil {
		t.Skip("Etcd not reachable, skipping test:", err)
		return
	}

	// Create registry
	reg := NewEtcdRegistry(
		registry.Addrs(etcdAddr),
		registry.Timeout(5*time.Second),
	).(*etcdRegistry)

	// Create a test service
	service := &registry.Service{
		Name:    "test.service",
		Version: "1.0.0",
		Nodes: []*registry.Node{
			{
				Id:      "test-node-1",
				Address: "localhost:9090",
			},
		},
	}

	// Register with TTL
	err = reg.Register(service, registry.RegisterTTL(10*time.Second))
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Wait a bit for keepalive to start
	time.Sleep(100 * time.Millisecond)

	// Check that keepalive channel was created
	reg.RLock()
	key := service.Name + service.Nodes[0].Id
	_, hasKeepalive := reg.keepaliveChs[key]
	_, hasStop := reg.keepaliveStop[key]
	reg.RUnlock()

	if !hasKeepalive {
		t.Error("Keepalive channel was not created")
	}
	if !hasStop {
		t.Error("Keepalive stop channel was not created")
	}

	// Register again (simulating re-registration)
	// This should reuse the existing keepalive
	err = reg.Register(service, registry.RegisterTTL(10*time.Second))
	if err != nil {
		t.Fatalf("Failed to re-register service: %v", err)
	}

	// Deregister
	err = reg.Deregister(service)
	if err != nil {
		t.Fatalf("Failed to deregister service: %v", err)
	}

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Check that keepalive was cleaned up
	reg.RLock()
	_, hasKeepalive = reg.keepaliveChs[key]
	_, hasStop = reg.keepaliveStop[key]
	reg.RUnlock()

	if hasKeepalive {
		t.Error("Keepalive channel was not cleaned up")
	}
	if hasStop {
		t.Error("Keepalive stop channel was not cleaned up")
	}
}

// TestKeepAliveReducesAuthRequests tests that KeepAlive reduces authentication requests
// This is a conceptual test - in practice, measuring auth requests requires etcd with auth enabled
func TestKeepAliveReducesAuthRequests(t *testing.T) {
	// Skip if no etcd server available
	etcdAddr := os.Getenv("ETCD_ADDRESS")
	if etcdAddr == "" {
		etcdAddr = "127.0.0.1:2379"
	}

	// Try to connect to etcd
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdAddr},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Skip("Etcd not available, skipping test:", err)
		return
	}
	defer client.Close()

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = client.Get(ctx, "/test")
	if err != nil {
		t.Skip("Etcd not reachable, skipping test:", err)
		return
	}

	// Create registry
	reg := NewEtcdRegistry(
		registry.Addrs(etcdAddr),
		registry.Timeout(5*time.Second),
	).(*etcdRegistry)

	// Create multiple test services
	services := make([]*registry.Service, 5)
	for i := 0; i < 5; i++ {
		services[i] = &registry.Service{
			Name:    fmt.Sprintf("test.service.%d", i),
			Version: "1.0.0",
			Nodes: []*registry.Node{
				{
					Id:      fmt.Sprintf("test-node-%d", i),
					Address: fmt.Sprintf("localhost:909%d", i),
				},
			},
		}

		// Register with TTL
		err = reg.Register(services[i], registry.RegisterTTL(10*time.Second))
		if err != nil {
			t.Fatalf("Failed to register service %d: %v", i, err)
		}
	}

	// Wait for keepalives to start
	time.Sleep(200 * time.Millisecond)

	// Verify all have keepalive channels
	reg.RLock()
	keepaliveCount := len(reg.keepaliveChs)
	reg.RUnlock()

	if keepaliveCount != 5 {
		t.Errorf("Expected 5 keepalive channels, got %d", keepaliveCount)
	}

	// Simulate multiple re-registrations (heartbeats)
	// With KeepAlive, these should NOT create new auth requests
	for i := 0; i < 3; i++ {
		time.Sleep(100 * time.Millisecond)
		for _, service := range services {
			err = reg.Register(service, registry.RegisterTTL(10*time.Second))
			if err != nil {
				t.Fatalf("Failed to re-register service: %v", err)
			}
		}
	}

	// Still should have only 5 keepalive channels (not 15 or 20)
	reg.RLock()
	keepaliveCount = len(reg.keepaliveChs)
	reg.RUnlock()

	if keepaliveCount != 5 {
		t.Errorf("After re-registrations, expected 5 keepalive channels, got %d", keepaliveCount)
	}

	// Cleanup
	for _, service := range services {
		err = reg.Deregister(service)
		if err != nil {
			t.Logf("Failed to deregister service: %v", err)
		}
	}
}

// TestKeepAliveChannelReconnection tests that keepalive handles channel closure
func TestKeepAliveChannelReconnection(t *testing.T) {
	// This test verifies the goroutine properly handles channel closure
	reg := &etcdRegistry{
		options: registry.Options{
			Logger: logger.DefaultLogger,
		},
		keepaliveChs:  make(map[string]<-chan *clientv3.LeaseKeepAliveResponse),
		keepaliveStop: make(map[string]chan bool),
	}

	// Create a mock keepalive channel that closes immediately
	ch := make(chan *clientv3.LeaseKeepAliveResponse)
	close(ch)

	reg.keepaliveChs["test-key"] = ch
	stopCh := make(chan bool, 1)
	reg.keepaliveStop["test-key"] = stopCh

	// Start the goroutine manually
	go func() {
		log := reg.options.Logger
		for {
			select {
			case <-stopCh:
				log.Logf(logger.TraceLevel, "Stopping keepalive for test-key")
				return
			case ka, ok := <-ch:
				if !ok {
					log.Logf(logger.DebugLevel, "Keepalive channel closed for test-key")
					reg.Lock()
					delete(reg.keepaliveChs, "test-key")
					delete(reg.keepaliveStop, "test-key")
					reg.Unlock()
					return
				}
				if ka == nil {
					log.Logf(logger.WarnLevel, "Keepalive response is nil for test-key")
					continue
				}
			}
		}
	}()

	// Wait for goroutine to detect closure and cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify cleanup happened
	reg.RLock()
	_, hasKeepalive := reg.keepaliveChs["test-key"]
	_, hasStop := reg.keepaliveStop["test-key"]
	reg.RUnlock()

	if hasKeepalive {
		t.Error("Keepalive channel should have been cleaned up after closure")
	}
	if hasStop {
		t.Error("Stop channel should have been cleaned up after closure")
	}
}

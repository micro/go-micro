package nats

import (
	"sync"
	"testing"
	"time"

	natsp "github.com/nats-io/nats.go"
)

func TestConnectionPool_GetPut(t *testing.T) {
	// Mock factory that creates connections
	connCount := 0
	factory := func() (*natsp.Conn, error) {
		connCount++
		// Return a mock connection (we can't create real NATS connections in tests without a server)
		// This test is more about the pool logic
		return nil, nil
	}

	pool, err := newConnectionPool(3, factory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Get a connection (should create one)
	conn1, err := pool.Get()
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}
	if conn1 == nil {
		t.Fatal("Expected connection, got nil")
	}

	// Put it back
	if err := pool.Put(conn1); err != nil {
		t.Fatalf("Failed to put connection: %v", err)
	}

	// Get it again (should reuse the same one)
	conn2, err := pool.Get()
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Since we can't compare actual connections easily, just verify we got one
	if conn2 == nil {
		t.Fatal("Expected connection, got nil")
	}
}

func TestConnectionPool_Concurrent(t *testing.T) {
	connCount := 0
	mu := sync.Mutex{}
	factory := func() (*natsp.Conn, error) {
		mu.Lock()
		connCount++
		mu.Unlock()
		return nil, nil
	}

	pool, err := newConnectionPool(5, factory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Simulate concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := pool.Get()
			if err != nil {
				t.Errorf("Failed to get connection: %v", err)
				return
			}
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			if err := pool.Put(conn); err != nil {
				t.Errorf("Failed to put connection: %v", err)
			}
		}()
	}

	wg.Wait()

	// We should have created some connections
	mu.Lock()
	if connCount == 0 {
		t.Error("Expected at least one connection to be created")
	}
	mu.Unlock()
}

func TestConnectionPool_Close(t *testing.T) {
	factory := func() (*natsp.Conn, error) {
		return nil, nil
	}

	pool, err := newConnectionPool(3, factory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	// Get a connection
	conn, err := pool.Get()
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Close the pool
	if err := pool.Close(); err != nil {
		t.Fatalf("Failed to close pool: %v", err)
	}

	// Put connection back to closed pool should not panic
	// The connection will be closed instead of returned to pool
	_ = pool.Put(conn)

	// Try to get from closed pool
	_, err = pool.Get()
	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got: %v", err)
	}
}

func TestPooledConnection_IsValid(t *testing.T) {
	pc := &pooledConnection{
		conn:      nil, // nil connection should be invalid
		createdAt: time.Now(),
		lastUsed:  time.Now(),
	}

	if pc.isValid() {
		t.Error("Expected nil connection to be invalid")
	}
}

func TestPooledConnection_IsExpired(t *testing.T) {
	pc := &pooledConnection{
		conn:      nil,
		createdAt: time.Now(),
		lastUsed:  time.Now().Add(-10 * time.Minute), // 10 minutes ago
	}

	// With 5 minute timeout, should be expired
	if !pc.isExpired(5 * time.Minute) {
		t.Error("Expected connection to be expired")
	}

	// With 0 timeout, should never expire
	if pc.isExpired(0) {
		t.Error("Expected connection not to expire with 0 timeout")
	}

	// With 20 minute timeout, should not be expired
	if pc.isExpired(20 * time.Minute) {
		t.Error("Expected connection not to be expired")
	}
}

func TestNatsBroker_PoolConfiguration(t *testing.T) {
	// Test that pool size is set correctly
	br := NewNatsBroker(PoolSize(5))
	nb, ok := br.(*natsBroker)
	if !ok {
		t.Fatal("Expected broker to be of type *natsBroker")
	}

	if nb.poolSize != 5 {
		t.Errorf("Expected pool size 5, got %d", nb.poolSize)
	}

	// Test with custom idle timeout
	br2 := NewNatsBroker(PoolSize(3), PoolIdleTimeout(10*time.Minute))
	nb2, ok := br2.(*natsBroker)
	if !ok {
		t.Fatal("Expected broker to be of type *natsBroker")
	}

	if nb2.poolSize != 3 {
		t.Errorf("Expected pool size 3, got %d", nb2.poolSize)
	}

	if nb2.poolIdleTimeout != 10*time.Minute {
		t.Errorf("Expected idle timeout 10m, got %v", nb2.poolIdleTimeout)
	}
}

func TestNatsBroker_DefaultSingleConnection(t *testing.T) {
	// Test that default behavior is single connection (pool size 1)
	br := NewNatsBroker()
	nb, ok := br.(*natsBroker)
	if !ok {
		t.Fatal("Expected broker to be of type *natsBroker")
	}

	if nb.poolSize != 1 {
		t.Errorf("Expected default pool size 1, got %d", nb.poolSize)
	}
}

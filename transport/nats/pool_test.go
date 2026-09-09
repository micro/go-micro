package nats

import (
	"sync"
	"testing"
	"time"

	natsp "github.com/nats-io/nats.go"
)

func TestTransportConnectionPool_GetPut(t *testing.T) {
	// Mock factory that creates connections
	connCount := 0
	factory := func() (*natsp.Conn, error) {
		connCount++
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

	if conn2 == nil {
		t.Fatal("Expected connection, got nil")
	}
}

func TestTransportConnectionPool_Concurrent(t *testing.T) {
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

func TestTransportConnectionPool_Close(t *testing.T) {
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
	_ = pool.Put(conn)

	// Try to get from closed pool
	_, err = pool.Get()
	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got: %v", err)
	}
}

func TestTransportPoolConfiguration(t *testing.T) {
	// Test with pool size 5
	tr := NewTransport(PoolSize(5))
	nt, ok := tr.(*ntport)
	if !ok {
		t.Fatal("Expected transport to be of type *ntport")
	}

	if nt.poolSize != 5 {
		t.Errorf("Expected pool size 5, got %d", nt.poolSize)
	}

	// Test with custom idle timeout
	tr2 := NewTransport(PoolSize(3), PoolIdleTimeout(10*time.Minute))
	nt2, ok := tr2.(*ntport)
	if !ok {
		t.Fatal("Expected transport to be of type *ntport")
	}

	if nt2.poolSize != 3 {
		t.Errorf("Expected pool size 3, got %d", nt2.poolSize)
	}

	if nt2.poolIdleTimeout != 10*time.Minute {
		t.Errorf("Expected idle timeout 10m, got %v", nt2.poolIdleTimeout)
	}
}

func TestTransportDefaultSingleConnection(t *testing.T) {
	// Test that default behavior is single connection (pool size 1)
	tr := NewTransport()
	nt, ok := tr.(*ntport)
	if !ok {
		t.Fatal("Expected transport to be of type *ntport")
	}

	if nt.poolSize != 1 {
		t.Errorf("Expected default pool size 1, got %d", nt.poolSize)
	}

	// With size 1, pool should not be created
	if nt.pool != nil {
		t.Error("Expected no pool with size 1")
	}
}

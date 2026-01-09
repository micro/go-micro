package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/registry"
)

// mockRegistry is a mock implementation of registry.Registry for testing
type mockRegistry struct {
	callCount int32
	delay     time.Duration
	err       error
	services  []*registry.Service
	mu        sync.Mutex
}

func (m *mockRegistry) Init(...registry.Option) error {
	return nil
}

func (m *mockRegistry) Options() registry.Options {
	return registry.Options{}
}

func (m *mockRegistry) Register(*registry.Service, ...registry.RegisterOption) error {
	return nil
}

func (m *mockRegistry) Deregister(*registry.Service, ...registry.DeregisterOption) error {
	return nil
}

func (m *mockRegistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	// Increment call count
	atomic.AddInt32(&m.callCount, 1)

	// Simulate delay (e.g., network latency)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// Return error if configured
	if m.err != nil {
		return nil, m.err
	}

	// Return services
	return m.services, nil
}

func (m *mockRegistry) ListServices(...registry.ListOption) ([]*registry.Service, error) {
	return nil, nil
}

func (m *mockRegistry) Watch(...registry.WatchOption) (registry.Watcher, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRegistry) String() string {
	return "mock"
}

func (m *mockRegistry) getCallCount() int32 {
	return atomic.LoadInt32(&m.callCount)
}

// TestSingleflightPreventsStampede verifies that concurrent requests for the same service
// only result in a single call to the underlying registry
func TestSingleflightPreventsStampede(t *testing.T) {
	mock := &mockRegistry{
		delay: 100 * time.Millisecond, // Simulate slow etcd response
		services: []*registry.Service{
			{
				Name:    "test.service",
				Version: "1.0.0",
				Nodes: []*registry.Node{
					{Id: "node1", Address: "localhost:9090"},
				},
			},
		},
	}

	c := New(mock, func(o *Options) {
		o.TTL = time.Minute
		o.Logger = logger.DefaultLogger
	}).(*cache)

	// Launch 10 concurrent requests for the same service
	const concurrency = 10
	var wg sync.WaitGroup
	wg.Add(concurrency)

	results := make([][]*registry.Service, concurrency)
	errs := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			services, err := c.GetService("test.service")
			results[idx] = services
			errs[idx] = err
		}(i)
	}

	wg.Wait()

	// Verify that only 1 call was made to the underlying registry
	callCount := mock.getCallCount()
	if callCount != 1 {
		t.Errorf("Expected 1 call to registry, got %d", callCount)
	}

	// Verify all requests got the same result
	for i := 0; i < concurrency; i++ {
		if errs[i] != nil {
			t.Errorf("Request %d failed: %v", i, errs[i])
		}
		if len(results[i]) != 1 {
			t.Errorf("Request %d got %d services, expected 1", i, len(results[i]))
		}
	}
}

// TestSingleflightWithError verifies that when etcd fails, only one request is made
// and all concurrent callers receive the error
func TestSingleflightWithError(t *testing.T) {
	expectedErr := errors.New("etcd connection failed")
	mock := &mockRegistry{
		delay: 50 * time.Millisecond,
		err:   expectedErr,
	}

	c := New(mock, func(o *Options) {
		o.TTL = time.Minute
		o.Logger = logger.DefaultLogger
	}).(*cache)

	// Launch concurrent requests
	const concurrency = 10
	var wg sync.WaitGroup
	wg.Add(concurrency)

	errs := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			_, err := c.GetService("test.service")
			errs[idx] = err
		}(i)
	}

	wg.Wait()

	// Verify that only 1 call was made to the underlying registry
	callCount := mock.getCallCount()
	if callCount != 1 {
		t.Errorf("Expected 1 call to registry even on error, got %d", callCount)
	}

	// Verify all requests got the error
	for i := 0; i < concurrency; i++ {
		if errs[i] == nil {
			t.Errorf("Request %d should have failed", i)
		}
	}
}

// TestStaleCacheOnError verifies that stale cache is returned when registry fails
func TestStaleCacheOnError(t *testing.T) {
	mock := &mockRegistry{
		services: []*registry.Service{
			{
				Name:    "test.service",
				Version: "1.0.0",
				Nodes: []*registry.Node{
					{Id: "node1", Address: "localhost:9090"},
				},
			},
		},
	}

	c := New(mock, func(o *Options) {
		o.TTL = 100 * time.Millisecond // Short TTL for testing
		o.Logger = logger.DefaultLogger
	}).(*cache)

	// First request - should populate cache
	services, err := c.GetService("test.service")
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(services))
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Configure mock to fail
	mock.err = errors.New("etcd unavailable")

	// Second request - should return stale cache despite error
	services, err = c.GetService("test.service")
	if err != nil {
		t.Errorf("Should have returned stale cache, got error: %v", err)
	}
	if len(services) != 1 {
		t.Errorf("Expected stale cache with 1 service, got %d", len(services))
	}
}

// TestCachePenetrationPrevention verifies the complete flow:
// 1. Cache populated
// 2. Cache expires
// 3. Registry fails
// 4. Concurrent requests don't stampede registry
// 5. Stale cache is returned
func TestCachePenetrationPrevention(t *testing.T) {
	mock := &mockRegistry{
		services: []*registry.Service{
			{
				Name:    "test.service",
				Version: "1.0.0",
				Nodes: []*registry.Node{
					{Id: "node1", Address: "localhost:9090"},
				},
			},
		},
	}

	c := New(mock, func(o *Options) {
		o.TTL = 100 * time.Millisecond
		o.Logger = logger.DefaultLogger
	}).(*cache)

	// Initial request to populate cache
	_, err := c.GetService("test.service")
	if err != nil {
		t.Fatalf("Initial request failed: %v", err)
	}

	initialCalls := mock.getCallCount()
	if initialCalls != 1 {
		t.Fatalf("Expected 1 initial call, got %d", initialCalls)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Configure mock to fail with delay
	mock.err = errors.New("etcd overloaded")
	mock.delay = 100 * time.Millisecond

	// Launch many concurrent requests (simulating stampede)
	const concurrency = 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	successCount := int32(0)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			services, err := c.GetService("test.service")
			// Should return stale cache without error
			if err == nil && len(services) > 0 {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	// Verify:
	// 1. Only ONE additional call was made (singleflight prevented stampede)
	totalCalls := mock.getCallCount()
	if totalCalls != 2 { // initial + 1 retry
		t.Errorf("Expected 2 total calls (1 initial + 1 retry), got %d", totalCalls)
	}

	// 2. All requests got stale cache (no errors)
	if successCount != concurrency {
		t.Errorf("Expected all %d requests to succeed with stale cache, got %d", concurrency, successCount)
	}
}

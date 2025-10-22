package server

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/registry"
)

// TestSubscriberNoDuplicates verifies that when multiple subscribers are registered
// for the same topic with different queues, each handler is called exactly once
// per published message (no duplicate deliveries).
func TestSubscriberNoDuplicates(t *testing.T) {
	// Create a memory broker
	memBroker := broker.NewMemoryBroker()
	if err := memBroker.Connect(); err != nil {
		t.Fatalf("Failed to connect broker: %v", err)
	}
	defer memBroker.Disconnect()

	// Create a memory registry
	memRegistry := registry.NewMemoryRegistry()

	// Create server with memory broker and registry
	srv := NewRPCServer(
		Broker(memBroker),
		Registry(memRegistry),
		Name("test.service"),
		Id("test-1"),
		Address("127.0.0.1:0"),
	)

	// Track handler invocations
	var countA, countB, countC int32

	// Handler functions
	handlerA := func(ctx context.Context, msg *TestMessage) error {
		atomic.AddInt32(&countA, 1)
		return nil
	}

	handlerB := func(ctx context.Context, msg *TestMessage) error {
		atomic.AddInt32(&countB, 1)
		return nil
	}

	handlerC := func(ctx context.Context, msg *TestMessage) error {
		atomic.AddInt32(&countC, 1)
		return nil
	}

	// Register three subscribers with same topic but different queues
	topic := "EVENT_1"

	subA := srv.NewSubscriber(topic, handlerA, SubscriberQueue("A"))
	if err := srv.Subscribe(subA); err != nil {
		t.Fatalf("Failed to subscribe A: %v", err)
	}

	subB := srv.NewSubscriber(topic, handlerB, SubscriberQueue("B"))
	if err := srv.Subscribe(subB); err != nil {
		t.Fatalf("Failed to subscribe B: %v", err)
	}

	subC := srv.NewSubscriber(topic, handlerC, SubscriberQueue("C"))
	if err := srv.Subscribe(subC); err != nil {
		t.Fatalf("Failed to subscribe C: %v", err)
	}

	// Start the server (this will trigger reSubscribe)
	if err := srv.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer srv.Stop()

	// Give server time to establish subscriptions
	time.Sleep(100 * time.Millisecond)

	// Publish a message to the topic
	if err := memBroker.Publish(topic, &broker.Message{
		Header: map[string]string{
			"Micro-Topic": topic,
			"Content-Type": "application/json",
		},
		Body: []byte(`{"value":"test"}`),
	}); err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// Give handlers time to process
	time.Sleep(200 * time.Millisecond)

	// Verify each handler was called exactly once
	if got := atomic.LoadInt32(&countA); got != 1 {
		t.Errorf("Handler A called %d times, expected 1", got)
	}
	if got := atomic.LoadInt32(&countB); got != 1 {
		t.Errorf("Handler B called %d times, expected 1", got)
	}
	if got := atomic.LoadInt32(&countC); got != 1 {
		t.Errorf("Handler C called %d times, expected 1", got)
	}
}

// TestSubscriberMultipleTopics verifies that subscribers for different topics
// each receive their respective messages correctly.
func TestSubscriberMultipleTopics(t *testing.T) {
	// Create a memory broker
	memBroker := broker.NewMemoryBroker()
	if err := memBroker.Connect(); err != nil {
		t.Fatalf("Failed to connect broker: %v", err)
	}
	defer memBroker.Disconnect()

	// Create a memory registry
	memRegistry := registry.NewMemoryRegistry()

	// Create server
	srv := NewRPCServer(
		Broker(memBroker),
		Registry(memRegistry),
		Name("test.service"),
		Id("test-2"),
		Address("127.0.0.1:0"),
	)

	// Track handler invocations
	var count1, count2 int32
	var wg sync.WaitGroup
	wg.Add(2)

	// Handler functions
	handler1 := func(ctx context.Context, msg *TestMessage) error {
		atomic.AddInt32(&count1, 1)
		wg.Done()
		return nil
	}

	handler2 := func(ctx context.Context, msg *TestMessage) error {
		atomic.AddInt32(&count2, 1)
		wg.Done()
		return nil
	}

	// Register subscribers for different topics
	topic1 := "TOPIC_1"
	topic2 := "TOPIC_2"

	sub1 := srv.NewSubscriber(topic1, handler1)
	if err := srv.Subscribe(sub1); err != nil {
		t.Fatalf("Failed to subscribe to topic1: %v", err)
	}

	sub2 := srv.NewSubscriber(topic2, handler2)
	if err := srv.Subscribe(sub2); err != nil {
		t.Fatalf("Failed to subscribe to topic2: %v", err)
	}

	// Start the server
	if err := srv.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer srv.Stop()

	// Give server time to establish subscriptions
	time.Sleep(100 * time.Millisecond)

	// Publish messages to different topics
	if err := memBroker.Publish(topic1, &broker.Message{
		Header: map[string]string{
			"Micro-Topic": topic1,
			"Content-Type": "application/json",
		},
		Body: []byte(`{"value":"test1"}`),
	}); err != nil {
		t.Fatalf("Failed to publish to topic1: %v", err)
	}

	if err := memBroker.Publish(topic2, &broker.Message{
		Header: map[string]string{
			"Micro-Topic": topic2,
			"Content-Type": "application/json",
		},
		Body: []byte(`{"value":"test2"}`),
	}); err != nil {
		t.Fatalf("Failed to publish to topic2: %v", err)
	}

	// Wait for handlers to be called
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for handlers to be called")
	}

	// Verify each handler was called exactly once
	if got := atomic.LoadInt32(&count1); got != 1 {
		t.Errorf("Handler 1 called %d times, expected 1", got)
	}
	if got := atomic.LoadInt32(&count2); got != 1 {
		t.Errorf("Handler 2 called %d times, expected 1", got)
	}
}

// TestMessage is a test message type
type TestMessage struct {
	Value string `json:"value"`
}

package web

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSEBroadcaster_Basic(t *testing.T) {
	// Create broadcaster
	b := NewSSEBroadcaster()
	if err := b.Start(); err != nil {
		t.Fatalf("Failed to start broadcaster: %v", err)
	}
	defer b.Stop()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(b.Handler()))
	defer server.Close()

	// Connect client
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer resp.Body.Close()

	// Check headers
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %s", ct)
	}

	// Read initial connection event
	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if !strings.HasPrefix(line, "event: connected") {
		t.Errorf("Expected connected event, got: %s", line)
	}
}

func TestSSEBroadcaster_BroadcastEvent(t *testing.T) {
	b := NewSSEBroadcaster()
	if err := b.Start(); err != nil {
		t.Fatalf("Failed to start broadcaster: %v", err)
	}
	defer b.Stop()

	server := httptest.NewServer(http.HandlerFunc(b.Handler()))
	defer server.Close()

	// Connect client
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer resp.Body.Close()

	// Wait for client to register
	time.Sleep(50 * time.Millisecond)

	// Broadcast an event
	testData := map[string]string{"message": "hello"}
	if err := b.BroadcastEvent("test", testData); err != nil {
		t.Fatalf("Failed to broadcast: %v", err)
	}

	// Read and verify
	reader := bufio.NewReader(resp.Body)
	
	// Skip connection event
	for i := 0; i < 3; i++ {
		reader.ReadString('\n')
	}

	// Read broadcast event
	line, _ := reader.ReadString('\n')
	if !strings.HasPrefix(line, "data:") {
		t.Errorf("Expected data line, got: %s", line)
	}

	// Parse the data
	dataStr := strings.TrimPrefix(line, "data: ")
	dataStr = strings.TrimSpace(dataStr)
	
	var event SSEEvent
	if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	if event.Event != "test" {
		t.Errorf("Expected event type 'test', got '%s'", event.Event)
	}
}

func TestSSEBroadcaster_ClientCount(t *testing.T) {
	b := NewSSEBroadcaster()
	if err := b.Start(); err != nil {
		t.Fatalf("Failed to start broadcaster: %v", err)
	}
	defer b.Stop()

	server := httptest.NewServer(http.HandlerFunc(b.Handler()))
	defer server.Close()

	if count := b.ClientCount(); count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}

	// Connect a client
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	if count := b.ClientCount(); count != 1 {
		t.Errorf("Expected 1 client, got %d", count)
	}

	resp.Body.Close()

	// Wait for unregistration
	time.Sleep(50 * time.Millisecond)

	if count := b.ClientCount(); count != 0 {
		t.Errorf("Expected 0 clients after disconnect, got %d", count)
	}
}

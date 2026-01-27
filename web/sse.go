// Package web provides a web service for go-micro
package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go-micro.dev/v5/events"
	log "go-micro.dev/v5/logger"
)

// SSEClient represents a connected SSE client
type SSEClient struct {
	id       string
	send     chan []byte
	done     chan struct{}
	metadata map[string]string
}

// SSEBroadcaster manages SSE connections and broadcasts events to connected clients
type SSEBroadcaster struct {
	clients    map[*SSEClient]struct{}
	register   chan *SSEClient
	unregister chan *SSEClient
	broadcast  chan []byte
	stream     events.Stream
	topics     []string
	logger     log.Logger
	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
}

// SSEEvent represents an event to be sent to clients
type SSEEvent struct {
	ID    string      `json:"id,omitempty"`
	Event string      `json:"event,omitempty"`
	Data  interface{} `json:"data"`
}

// SSEOption is a function that configures the SSEBroadcaster
type SSEOption func(*SSEBroadcaster)

// WithStream sets the events stream for the broadcaster
func WithStream(stream events.Stream) SSEOption {
	return func(b *SSEBroadcaster) {
		b.stream = stream
	}
}

// WithTopics sets the topics to subscribe to
func WithTopics(topics ...string) SSEOption {
	return func(b *SSEBroadcaster) {
		b.topics = topics
	}
}

// WithSSELogger sets the logger for the broadcaster
func WithSSELogger(logger log.Logger) SSEOption {
	return func(b *SSEBroadcaster) {
		b.logger = logger
	}
}

// NewSSEBroadcaster creates a new SSE broadcaster
func NewSSEBroadcaster(opts ...SSEOption) *SSEBroadcaster {
	b := &SSEBroadcaster{
		clients:    make(map[*SSEClient]struct{}),
		register:   make(chan *SSEClient),
		unregister: make(chan *SSEClient),
		broadcast:  make(chan []byte, 256),
		logger:     log.DefaultLogger,
		stopCh:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Start begins the broadcaster's event loop and subscribes to configured topics
func (b *SSEBroadcaster) Start() error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return nil
	}
	b.running = true
	b.mu.Unlock()

	// Start the main event loop
	go b.run()

	// Subscribe to topics if stream is configured
	if b.stream != nil && len(b.topics) > 0 {
		for _, topic := range b.topics {
			if err := b.subscribeToTopic(topic); err != nil {
				b.logger.Logf(log.ErrorLevel, "Failed to subscribe to topic %s: %v", topic, err)
			}
		}
	}

	return nil
}

// Stop gracefully shuts down the broadcaster
func (b *SSEBroadcaster) Stop() {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return
	}
	b.running = false
	b.mu.Unlock()

	close(b.stopCh)
}

func (b *SSEBroadcaster) run() {
	for {
		select {
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = struct{}{}
			b.mu.Unlock()
			b.logger.Logf(log.DebugLevel, "SSE client connected: %s", client.id)

		case client := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				close(client.send)
			}
			b.mu.Unlock()
			b.logger.Logf(log.DebugLevel, "SSE client disconnected: %s", client.id)

		case message := <-b.broadcast:
			b.mu.RLock()
			for client := range b.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full, skip
				}
			}
			b.mu.RUnlock()

		case <-b.stopCh:
			b.mu.Lock()
			for client := range b.clients {
				close(client.send)
				delete(b.clients, client)
			}
			b.mu.Unlock()
			return
		}
	}
}

func (b *SSEBroadcaster) subscribeToTopic(topic string) error {
	eventChan, err := b.stream.Consume(topic)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event := <-eventChan:
				b.Broadcast(event.Payload)
			case <-b.stopCh:
				return
			}
		}
	}()

	b.logger.Logf(log.InfoLevel, "SSE broadcaster subscribed to topic: %s", topic)
	return nil
}

// Broadcast sends a message to all connected clients
func (b *SSEBroadcaster) Broadcast(data []byte) {
	select {
	case b.broadcast <- data:
	default:
		b.logger.Log(log.WarnLevel, "SSE broadcast channel full, dropping message")
	}
}

// BroadcastEvent sends a structured event to all connected clients
func (b *SSEBroadcaster) BroadcastEvent(eventType string, data interface{}) error {
	event := SSEEvent{
		ID:    fmt.Sprintf("%d", time.Now().UnixNano()),
		Event: eventType,
		Data:  data,
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	b.Broadcast(jsonData)
	return nil
}

// BroadcastHTML sends raw HTML to clients (for htmx/datastar integration)
func (b *SSEBroadcaster) BroadcastHTML(eventType string, html string) {
	// Format as SSE with event type for htmx sse-swap
	message := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, html)
	b.Broadcast([]byte(message))
}

// ClientCount returns the number of connected clients
func (b *SSEBroadcaster) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Handler returns an http.HandlerFunc for SSE connections
func (b *SSEBroadcaster) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the client supports SSE
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

		// Create client
		client := &SSEClient{
			id:   fmt.Sprintf("%d", time.Now().UnixNano()),
			send: make(chan []byte, 64),
			done: make(chan struct{}),
		}

		// Register client
		b.register <- client

		// Ensure cleanup on disconnect
		defer func() {
			b.unregister <- client
		}()

		// Send initial connection event
		fmt.Fprintf(w, "event: connected\ndata: {\"id\":\"%s\"}\n\n", client.id)
		flusher.Flush()

		// Keep-alive ticker
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case message, ok := <-client.send:
				if !ok {
					return
				}
				// Check if message is already SSE formatted (contains "event:" or "data:")
				if len(message) > 0 && (message[0] == 'e' || message[0] == 'd') {
					w.Write(message)
				} else {
					fmt.Fprintf(w, "data: %s\n\n", message)
				}
				flusher.Flush()

			case <-ticker.C:
				// Send keep-alive comment
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()

			case <-r.Context().Done():
				return
			}
		}
	}
}

// GinHandler returns a handler compatible with Gin framework
func (b *SSEBroadcaster) GinHandler() interface{} {
	return b.Handler()
}

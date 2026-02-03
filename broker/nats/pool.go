package nats

import (
	"errors"
	"sync"
	"time"

	natsp "github.com/nats-io/nats.go"
)

var (
	// ErrPoolExhausted is returned when no connections are available in the pool
	ErrPoolExhausted = errors.New("connection pool exhausted")
	// ErrPoolClosed is returned when trying to use a closed pool
	ErrPoolClosed = errors.New("connection pool is closed")
)

// connectionPool manages a pool of NATS connections
type connectionPool struct {
	mu          sync.RWMutex
	connections chan *pooledConnection
	factory     func() (*natsp.Conn, error)
	size        int
	maxIdle     int
	idleTimeout time.Duration
	closed      bool
}

// pooledConnection wraps a NATS connection with metadata
type pooledConnection struct {
	conn      *natsp.Conn
	createdAt time.Time
	lastUsed  time.Time
	mu        sync.Mutex
}

// newConnectionPool creates a new connection pool
func newConnectionPool(size int, factory func() (*natsp.Conn, error)) (*connectionPool, error) {
	if size <= 0 {
		size = 1
	}

	pool := &connectionPool{
		connections: make(chan *pooledConnection, size),
		factory:     factory,
		size:        size,
		maxIdle:     size,
		idleTimeout: 5 * time.Minute,
		closed:      false,
	}

	return pool, nil
}

// Get retrieves a connection from the pool or creates a new one
func (p *connectionPool) Get() (*pooledConnection, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, ErrPoolClosed
	}
	p.mu.RUnlock()

	// Try to get an existing connection from the pool
	select {
	case conn := <-p.connections:
		// Check if connection is still valid and not idle for too long
		if conn.isValid() && !conn.isExpired(p.idleTimeout) {
			conn.lastUsed = time.Now()
			return conn, nil
		}
		// Connection is invalid or expired, close it and create a new one
		conn.close()
		return p.createConnection()
	default:
		// No connection available, create a new one
		return p.createConnection()
	}
}

// Put returns a connection to the pool
func (p *connectionPool) Put(conn *pooledConnection) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return conn.close()
	}

	// Check if connection is still valid
	if !conn.isValid() {
		return conn.close()
	}

	conn.lastUsed = time.Now()

	// Try to return connection to pool
	select {
	case p.connections <- conn:
		return nil
	default:
		// Pool is full, close the connection
		return conn.close()
	}
}

// Close closes all connections in the pool
func (p *connectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	close(p.connections)

	// Close all connections in the pool
	for conn := range p.connections {
		conn.close()
	}

	return nil
}

// createConnection creates a new pooled connection
func (p *connectionPool) createConnection() (*pooledConnection, error) {
	conn, err := p.factory()
	if err != nil {
		return nil, err
	}

	return &pooledConnection{
		conn:      conn,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
	}, nil
}

// isValid checks if the underlying NATS connection is valid
func (pc *pooledConnection) isValid() bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.conn == nil {
		return false
	}

	status := pc.conn.Status()
	return status == natsp.CONNECTED || status == natsp.RECONNECTING
}

// isExpired checks if the connection has been idle for too long
func (pc *pooledConnection) isExpired(timeout time.Duration) bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if timeout <= 0 {
		return false
	}

	return time.Since(pc.lastUsed) > timeout
}

// close closes the underlying NATS connection
func (pc *pooledConnection) close() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.conn != nil {
		pc.conn.Close()
		pc.conn = nil
	}
	return nil
}

// Conn returns the underlying NATS connection
func (pc *pooledConnection) Conn() *natsp.Conn {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.conn
}

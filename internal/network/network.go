// Package network is a process-local registry of server dispatchers — the
// neutral seam an in-process client fast-path uses to reach a server running in
// the same process without going over the network transport.
//
// It lives in internal/ and speaks only in transport.Message so neither the
// client nor the server package has to import the other: a running server
// registers a Handler under its service name; an opted-in client looks one up
// and dispatches directly, skipping dial, codec-over-socket, and the transport
// pump. Nothing here runs unless a server registers and a client opts in.
package network

import (
	"context"
	"sync"

	"go-micro.dev/v6/transport"
)

// Handler dispatches one request against a process-local server's handler
// table and returns the reply. req and the returned message carry the same
// codec-encoded body + headers the transport would have carried.
type Handler func(ctx context.Context, req *transport.Message) (*transport.Message, error)

var (
	mu  sync.RWMutex
	reg = map[string]Handler{}
)

// Register makes service reachable in-process via h. A server calls this when
// it starts; calling again replaces the handler.
func Register(service string, h Handler) {
	mu.Lock()
	reg[service] = h
	mu.Unlock()
}

// Deregister removes service's in-process handler. A server calls this when it
// stops, so a later in-process call falls back to the network path.
func Deregister(service string) {
	mu.Lock()
	delete(reg, service)
	mu.Unlock()
}

// Lookup returns the in-process handler for service, if one is registered.
func Lookup(service string) (Handler, bool) {
	mu.RLock()
	h, ok := reg[service]
	mu.RUnlock()
	return h, ok
}

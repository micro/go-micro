// Package proxy is a transparent proxy built on the go-micro/server
package proxy

import (
	"context"

	"github.com/micro/go-micro/v2/server"
)

// Proxy can be used as a proxy server for go-micro services
type Proxy interface {
	// ProcessMessage handles inbound messages
	ProcessMessage(context.Context, server.Message) error
	// ServeRequest handles inbound requests
	ServeRequest(context.Context, server.Request, server.Response) error
	// Name of the proxy protocol
	String() string
}

var (
	DefaultEndpoint = "localhost:9090"
)

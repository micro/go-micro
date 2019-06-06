// Package proxy is a transparent proxy built on the go-micro/server
package proxy

import (
	"context"

	"github.com/micro/go-micro/options"
	"github.com/micro/go-micro/server"
)

// Proxy can be used as a proxy server for go-micro services
type Proxy interface {
	options.Options
	// ServeRequest will serve a request
	ServeRequest(context.Context, server.Request, server.Response) error
	// run the proxy
	Run() error
}

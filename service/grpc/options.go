package grpc

import (
	"crypto/tls"

	gc "github.com/micro/go-micro/v3/client/grpc"
	gs "github.com/micro/go-micro/v3/server/grpc"
	"github.com/micro/go-micro/v3/service"
)

// WithTLS sets the TLS config for the service
func WithTLS(t *tls.Config) service.Option {
	return func(o *service.Options) {
		o.Client.Init(
			gc.AuthTLS(t),
		)
		o.Server.Init(
			gs.AuthTLS(t),
		)
	}
}

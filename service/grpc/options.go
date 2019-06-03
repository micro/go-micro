package grpc

import (
	"crypto/tls"

	"github.com/micro/go-micro"
	gc "github.com/micro/go-plugins/client/grpc"
	gs "github.com/micro/go-plugins/server/grpc"
)

// WithTLS sets the TLS config for the service
func WithTLS(t *tls.Config) micro.Option {
	return func(o *micro.Options) {
		o.Client.Init(
			gc.AuthTLS(t),
		)
		o.Server.Init(
			gs.AuthTLS(t),
		)
	}
}

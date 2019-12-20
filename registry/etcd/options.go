package etcd

import (
	"context"

	"github.com/micro/go-micro/registry"
	"go.uber.org/zap"
	"google.golang.org/grpc/grpclog"
)

type authKey struct{}

type logConfigKey struct{}

type logSetKey struct{}

type authCreds struct {
	Username string
	Password string
}

// Auth allows you to specify username/password
func Auth(username, password string) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, authKey{}, &authCreds{Username: username, Password: password})
	}
}

// LogConfig allows you to set etcd log config
func LogConfig(config *zap.Config) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, logConfigKey{}, config)
	}
}

// LogSet allows you to set etcd grpc log config
// LogSet is different from LogConfig. LogSet set the grpc communicate log between etcd client and server.
// LogConfig set the log in etcd client
func LogSet(l grpclog.LoggerV2) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, logSetKey{}, l)
	}
}

package nacos


import (
	"context"

	"go-micro.dev/v4/registry"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

type addressKey struct{}
type configKey struct{}

// WithAddress sets the nacos address
func WithAddress(addrs []string) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, addressKey{}, addrs)
	}
}

// WithClientConfig sets the nacos config
func WithClientConfig(cc constant.ClientConfig) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, configKey{}, cc)
	}
}


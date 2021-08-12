package nacos

import (
	"context"

	"github.com/asim/go-micro/v3/config/source"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

type addressKey struct{}
type configKey struct{}
type groupKey struct{}
type dataIdKey struct{}
type encoderKey struct{}

// WithAddress sets the nacos address
func WithAddress(addrs []string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, addressKey{}, addrs)
	}
}

// WithClientConfig sets the nacos config
func WithClientConfig(cc constant.ClientConfig) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, configKey{}, cc)
	}
}

// WithGroup sets nacos config group
func WithGroup(g string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, groupKey{}, g)
	}
}

// WithDataId sets nacos config dataId
func WithDataId(id string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, dataIdKey{}, id)
	}
}

package oauth

import (
	"github.com/micro/go-micro/v2/auth/provider"
)

// NewProvider returns an initialised oauth provider
func NewProvider(opts ...provider.Option) provider.Provider {
	var options provider.Options
	for _, o := range opts {
		o(&options)
	}
	return &oauth{options}
}

type oauth struct {
	opts provider.Options
}

func (o *oauth) String() string {
	return "oauth"
}

func (o *oauth) Options() provider.Options {
	return o.opts
}

func (o *oauth) Endpoint() string {
	return o.opts.Endpoint
}

func (o *oauth) Redirect() string {
	return o.opts.Redirect
}

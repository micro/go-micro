package basic

import (
	"github.com/micro/go-micro/v2/auth/provider"
)

// NewProvider returns an initialised basic provider
func NewProvider(opts ...provider.Option) provider.Provider {
	var options provider.Options
	for _, o := range opts {
		o(&options)
	}
	return &basic{options}
}

type basic struct {
	opts provider.Options
}

func (b *basic) String() string {
	return "basic"
}

func (b *basic) Options() provider.Options {
	return b.opts
}

func (b *basic) Endpoint(...provider.EndpointOption) string {
	return ""
}

func (b *basic) Redirect() string {
	return ""
}

package test

import (
	"github.com/micro/go-micro/v2/auth/provider"
)

func NewProvider(opts ...provider.Option) provider.Provider {
	return new(test)
}

type test struct{}

func (t *test) Type() string {
	return "test"
}

func (t *test) Options() provider.Options {
	return *new(provider.Options)
}

func (t *test) Endpoint() string {
	return ""
}

func (t *test) Redirect() string {
	return ""
}

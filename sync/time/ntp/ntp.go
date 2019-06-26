// Package ntp provides ntp synchronized time
package ntp

import (
	"context"
	gotime "time"

	"github.com/beevik/ntp"
	"github.com/micro/go-micro/sync/time"
)

type ntpTime struct {
	server string
}

type ntpServerKey struct{}

func (n *ntpTime) Now() (gotime.Time, error) {
	return ntp.Time(n.server)
}

// NewTime returns ntp time
func NewTime(opts ...time.Option) time.Time {
	options := time.Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	server := "time.google.com"

	if k, ok := options.Context.Value(ntpServerKey{}).(string); ok {
		server = k
	}

	return &ntpTime{
		server: server,
	}
}

// WithServer sets the ntp server
func WithServer(s string) time.Option {
	return func(o *time.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, ntpServerKey{}, s)
	}
}

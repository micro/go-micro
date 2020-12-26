package awsxray

import (
	"github.com/aws/aws-sdk-go/service/xray"
)

type Options struct {
	// XRay Client when using API
	Client *xray.XRay
	// Daemon address when using UDP
	Daemon string
	// Name of segments e.g the service
	Name string
}

type Option func(o *Options)

// WithName sets the name used for each segment. Use Service Name
func WithName(name string) Option {
	return func(o *Options) {
		o.Name = name
	}
}

// WithClient sets the XRay Client to use to send segments
func WithClient(x *xray.XRay) Option {
	return func(o *Options) {
		o.Client = x
	}
}

// WithDaemon sets the address of the XRay Daemon to send segements
func WithDaemon(addr string) Option {
	return func(o *Options) {
		o.Daemon = addr
	}
}

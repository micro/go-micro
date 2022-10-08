package transport

import (
	"context"
	"net"
)

type netListener struct{}

// setTransportOption Set option for transport
func setTransportOption(k, v interface{}) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

// getListener Get net.Listener from ListenOptions
func getListener(o *Options) net.Listener {
	if o.Context == nil {
		return nil
	}

	if l, ok := o.Context.Value(netListener{}).(net.Listener); ok && l != nil {
		return l
	}

	return nil
}

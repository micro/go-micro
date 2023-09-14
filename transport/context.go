package transport

import (
	"net"
)

type netListener struct{}

// getNetListener Get net.Listener from ListenOptions.
func getNetListener(o *ListenOptions) net.Listener {
	if o.Context == nil {
		return nil
	}

	if l, ok := o.Context.Value(netListener{}).(net.Listener); ok && l != nil {
		return l
	}

	return nil
}

package client

/*
Wrapper is a type of middleware for the go-micro client. It allows
the client to be "wrapped" so that requests and responses can be intercepted
to perform extra requirements such as auth, tracing, monitoring, logging, etc.

Example usage:

	import (
		"log"
		"github.com/micro/go-micro/client"

	)

	type LogWrapper struct {
		client.Client
	}

	func (l *LogWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
		log.Println("Making request to service " + req.Service() + " method " + req.Method())
		return w.Client.Call(ctx, req, rsp)
	}

	func Wrapper(c client.Client) client.Client {
		return &LogWrapper{c}
	}

	func main() {
		c := client.NewClient(client.Wrap(Wrapper))

	}


*/

import (
	"golang.org/x/net/context"
)

// CallFunc represents the individual call func
type CallFunc func(ctx context.Context, address string, req Request, rsp interface{}, opts CallOptions) error

// CallWrapper is a low level wrapper for the CallFunc
type CallWrapper func(CallFunc) CallFunc

// Wrapper wraps a client and returns a client
type Wrapper func(Client) Client

// StreamWrapper wraps a Stream and returns the equivalent
type StreamWrapper func(Streamer) Streamer

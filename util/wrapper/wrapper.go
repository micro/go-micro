package wrapper

import (
	"context"
	"strings"

	"github.com/asim/go-micro/v3/auth"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/debug/stats"
	"github.com/asim/go-micro/v3/debug/trace"
	"github.com/asim/go-micro/v3/metadata"
	"github.com/asim/go-micro/v3/server"
)

type fromServiceWrapper struct {
	client.Client

	// headers to inject
	headers metadata.Metadata
}

var (
	HeaderPrefix = "Micro-"
)

func (f *fromServiceWrapper) setHeaders(ctx context.Context) context.Context {
	// don't overwrite keys
	return metadata.MergeContext(ctx, f.headers, false)
}

func (f *fromServiceWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	ctx = f.setHeaders(ctx)
	return f.Client.Call(ctx, req, rsp, opts...)
}

func (f *fromServiceWrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	ctx = f.setHeaders(ctx)
	return f.Client.Stream(ctx, req, opts...)
}

func (f *fromServiceWrapper) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	ctx = f.setHeaders(ctx)
	return f.Client.Publish(ctx, p, opts...)
}

// FromService wraps a client to inject service and auth metadata
func FromService(name string, c client.Client) client.Client {
	return &fromServiceWrapper{
		c,
		metadata.Metadata{
			HeaderPrefix + "From-Service": name,
		},
	}
}

// HandlerStats wraps a server handler to generate request/error stats
func HandlerStats(stats stats.Stats) server.HandlerWrapper {
	// return a handler wrapper
	return func(h server.HandlerFunc) server.HandlerFunc {
		// return a function that returns a function
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			// execute the handler
			err := h(ctx, req, rsp)
			// record the stats
			stats.Record(err)
			// return the error
			return err
		}
	}
}

type traceWrapper struct {
	client.Client

	name  string
	trace trace.Tracer
}

func (c *traceWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	newCtx, s := c.trace.Start(ctx, req.Service()+"."+req.Endpoint())

	s.Type = trace.SpanTypeRequestOutbound
	err := c.Client.Call(newCtx, req, rsp, opts...)
	if err != nil {
		s.Metadata["error"] = err.Error()
	}

	// finish the trace
	c.trace.Finish(s)

	return err
}

// TraceCall is a call tracing wrapper
func TraceCall(name string, t trace.Tracer, c client.Client) client.Client {
	return &traceWrapper{
		name:   name,
		trace:  t,
		Client: c,
	}
}

// TraceHandler wraps a server handler to perform tracing
func TraceHandler(t trace.Tracer) server.HandlerWrapper {
	// return a handler wrapper
	return func(h server.HandlerFunc) server.HandlerFunc {
		// return a function that returns a function
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			// don't store traces for debug
			if strings.HasPrefix(req.Endpoint(), "Debug.") {
				return h(ctx, req, rsp)
			}

			// get the span
			newCtx, s := t.Start(ctx, req.Service()+"."+req.Endpoint())
			s.Type = trace.SpanTypeRequestInbound

			err := h(newCtx, req, rsp)
			if err != nil {
				s.Metadata["error"] = err.Error()
			}

			// finish
			t.Finish(s)

			return err
		}
	}
}

func AuthCall(a func() auth.Auth, c client.Client) client.Client {
	return &authWrapper{Client: c, auth: a}
}

type authWrapper struct {
	client.Client
	auth func() auth.Auth
}

func (a *authWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	// parse the options
	var options client.CallOptions
	for _, o := range opts {
		o(&options)
	}

	// check to see if the authorization header has already been set.
	// We dont't override the header unless the ServiceToken option has
	// been specified or the header wasn't provided
	if _, ok := metadata.Get(ctx, "Authorization"); ok && !options.ServiceToken {
		return a.Client.Call(ctx, req, rsp, opts...)
	}

	// if auth is nil we won't be able to get an access token, so we execute
	// the request without one.
	aa := a.auth()
	if aa == nil {
		return a.Client.Call(ctx, req, rsp, opts...)
	}

	// set the namespace header if it has not been set (e.g. on a service to service request)
	if _, ok := metadata.Get(ctx, "Micro-Namespace"); !ok {
		ctx = metadata.Set(ctx, "Micro-Namespace", aa.Options().Namespace)
	}

	// check to see if we have a valid access token
	aaOpts := aa.Options()
	if aaOpts.Token != nil && !aaOpts.Token.Expired() {
		ctx = metadata.Set(ctx, "Authorization", auth.BearerScheme+aaOpts.Token.AccessToken)
		return a.Client.Call(ctx, req, rsp, opts...)
	}

	// call without an auth token
	return a.Client.Call(ctx, req, rsp, opts...)
}

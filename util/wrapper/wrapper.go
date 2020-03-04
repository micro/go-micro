package wrapper

import (
	"context"
	"strings"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/debug/stats"
	"github.com/micro/go-micro/v2/debug/trace"
	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/server"
)

type clientWrapper struct {
	client.Client

	// Auth interface
	auth func() auth.Auth
	// headers to inject
	headers metadata.Metadata
}

type traceWrapper struct {
	client.Client

	name  string
	trace trace.Tracer
}

var (
	HeaderPrefix = "Micro-"
	BearerScheme = "Bearer "
)

func (c *clientWrapper) setHeaders(ctx context.Context) context.Context {
	// copy metadata
	mda, _ := metadata.FromContext(ctx)
	md := metadata.Copy(mda)

	// get auth token
	if a := c.auth(); a != nil {
		tk := a.Options().Token
		// if the token if exists and auth header isn't set then set it
		if len(tk) > 0 && len(md["Authorization"]) == 0 {
			md["Authorization"] = BearerScheme + tk
		}
	}

	// set headers
	for k, v := range c.headers {
		if _, ok := md[k]; !ok {
			md[k] = v
		}
	}

	return metadata.NewContext(ctx, md)
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	ctx = c.setHeaders(ctx)
	return c.Client.Call(ctx, req, rsp, opts...)
}

func (c *clientWrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	ctx = c.setHeaders(ctx)
	return c.Client.Stream(ctx, req, opts...)
}

func (c *clientWrapper) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	ctx = c.setHeaders(ctx)
	return c.Client.Publish(ctx, p, opts...)
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

// FromService wraps a client to inject service and auth metadata
func FromService(name string, c client.Client, fn func() auth.Auth) client.Client {
	return &clientWrapper{
		c,
		fn,
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

// AuthHandler wraps a server handler to perform auth
func AuthHandler(fn func() auth.Auth) server.HandlerWrapper {
	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			// get the auth.Auth interface
			a := fn()

			// Check for debug endpoints which should be excluded from auth
			if strings.HasPrefix(req.Endpoint(), "Debug.") {
				return h(ctx, req, rsp)
			}

			// Extract the token if present. Note: if noop is being used
			// then the token can be blank without erroring
			var token string
			if header, ok := metadata.Get(ctx, "Authorization"); ok {
				// Ensure the correct scheme is being used
				if !strings.HasPrefix(header, BearerScheme) {
					return errors.Unauthorized("go.micro.auth", "invalid authorization header. expected Bearer schema")
				}

				token = header[len(BearerScheme):]
			}

			// Verify the token
			account, authErr := a.Verify(token)

			// If there is an account, set it in the context
			if authErr == nil {
				var err error
				ctx, err = auth.ContextWithAccount(ctx, account)

				if err != nil {
					return err
				}
			}

			// Return if the user disabled auth on this endpoint
			for _, e := range a.Options().Exclude {
				if e == req.Endpoint() {
					return h(ctx, req, rsp)
				}
			}

			// If the authErr is set, prevent the user from calling the endpoint
			if authErr != nil {
				return errors.Unauthorized("go.micro.auth", authErr.Error())
			}

			// The user is authorised, allow the call
			return h(ctx, req, rsp)
		}
	}
}

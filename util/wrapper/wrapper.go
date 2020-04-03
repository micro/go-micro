package wrapper

import (
	"context"
	"strings"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/debug/stats"
	"github.com/micro/go-micro/v2/debug/trace"
	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/logger"
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
)

func (c *clientWrapper) setHeaders(ctx context.Context) context.Context {
	// copy metadata
	mda, _ := metadata.FromContext(ctx)
	md := metadata.Copy(mda)

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

			// Check for auth service endpoints which should be excluded from auth
			if strings.HasPrefix(req.Endpoint(), "Auth.") {
				return h(ctx, req, rsp)
			}

			// Extract the token if present. Note: if noop is being used
			// then the token can be blank without erroring
			var token string
			if header, ok := metadata.Get(ctx, "Authorization"); ok {
				// Ensure the correct scheme is being used
				if !strings.HasPrefix(header, auth.BearerScheme) {
					return errors.Unauthorized(req.Service(), "invalid authorization header. expected Bearer schema")
				}

				token = header[len(auth.BearerScheme):]
			}

			// Get the namespace for the request
			namespace, ok := metadata.Get(ctx, auth.NamespaceKey)
			if !ok {
				logger.Errorf("Missing request namespace")
				namespace = auth.DefaultNamespace
			}

			// Inspect the token and get the account
			account, err := a.Inspect(token)
			if err != nil {
				account = &auth.Account{Namespace: auth.DefaultNamespace}
			}

			// Check the accounts namespace matches the namespace we're operating
			// within. If not forbid the request and log the occurance.
			if account.Namespace != namespace {
				logger.Warnf("Cross namespace request forbidden: account %v (%v) requested access to %v %v in the %v namespace",
					account.ID, account.Namespace, req.Service(), req.Endpoint(), namespace)
				return errors.Forbidden(req.Service(), "cross namespace request")
			}

			// construct the resource
			res := &auth.Resource{
				Type:      "service",
				Name:      req.Service(),
				Endpoint:  req.Endpoint(),
				Namespace: namespace,
			}

			// Verify the caller has access to the resource
			err = a.Verify(account, res)
			if err != nil && len(account.ID) > 0 {
				return errors.Forbidden(req.Service(), "Forbidden call made to %v:%v by %v", req.Service(), req.Endpoint(), account.ID)
			} else if err != nil {
				return errors.Unauthorized(req.Service(), "Unauthorised call made to %v:%v", req.Service(), req.Endpoint())
			}

			// There is an account, set it in the context
			ctx, err = auth.ContextWithAccount(ctx, account)
			if err != nil {
				return err
			}

			// The user is authorised, allow the call
			return h(ctx, req, rsp)
		}
	}
}

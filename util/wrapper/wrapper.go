package wrapper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/debug/stats"
	"github.com/micro/go-micro/v2/debug/trace"
	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/server"
	"github.com/micro/go-micro/v2/util/config"
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

type authWrapper struct {
	client.Client
	name string
	id   string
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
	if a == nil {
		return a.Client.Call(ctx, req, rsp, opts...)
	}

	// performs the call with the authorization token provided
	callWithToken := func(token string) error {
		ctx := metadata.Set(ctx, "Authorization", auth.BearerScheme+token)
		return a.Client.Call(ctx, req, rsp, opts...)
	}

	// check to see if we have a valid access token
	aaOpts := aa.Options()
	if aaOpts.Token != nil && aaOpts.Token.Expiry.Unix() > time.Now().Unix() {
		return callWithToken(aaOpts.Token.AccessToken)
	}

	// if we have a refresh token we can use this to generate another access token
	if aaOpts.Token != nil {
		tok, err := aa.Token(auth.WithToken(aaOpts.Token.RefreshToken))
		if err != nil {
			return err
		}
		aa.Init(auth.ClientToken(tok))
		return callWithToken(tok.AccessToken)
	}

	// if we have credentials we can generate a new token for the account
	if len(aaOpts.ID) > 0 && len(aaOpts.Secret) > 0 {
		tok, err := aa.Token(auth.WithCredentials(aaOpts.ID, aaOpts.Secret))
		if err != nil {
			return err
		}
		aa.Init(auth.ClientToken(tok))
		return callWithToken(tok.AccessToken)
	}

	// check to see if a token was provided in config, this is normally used for
	// setting the token when calling via the cli
	if token, err := config.Get("micro", "auth", "token"); err == nil && len(token) > 0 {
		return callWithToken(token)
	}

	// determine the type of service from the name. we do this so we can allocate
	// different roles depending on the type of services. e.g. we don't want web
	// services talking directly to the runtime. TODO: find a better way to determine
	// the type of service
	serviceType := "service"
	if strings.Contains(a.name, "api") {
		serviceType = "api"
	} else if strings.Contains(a.name, "web") {
		serviceType = "web"
	}

	// generate a new auth account for the service
	name := fmt.Sprintf("%v-%v", a.name, a.id)
	acc, err := aa.Generate(name, auth.WithNamespace(aaOpts.Namespace), auth.WithRoles(serviceType))
	if err != nil {
		return err
	}
	token, err := aa.Token(auth.WithCredentials(acc.ID, acc.Secret))
	if err != nil {
		return err
	}
	aa.Init(auth.ClientToken(token))

	// use the token to execute the request
	return callWithToken(token.AccessToken)
}

// AuthClient wraps requests with the auth header
func AuthClient(name string, id string, auth func() auth.Auth, c client.Client) client.Client {
	return &authWrapper{c, name, id, auth}
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
				if !strings.HasPrefix(header, auth.BearerScheme) {
					return errors.Unauthorized(req.Service(), "invalid authorization header. expected Bearer schema")
				}

				token = header[len(auth.BearerScheme):]
			}

			// Inspect the token and get the account
			account, err := a.Inspect(token)
			if err != nil {
				account = &auth.Account{Namespace: a.Options().Namespace}
			}

			// construct the resource
			res := &auth.Resource{
				Type:     "service",
				Name:     req.Service(),
				Endpoint: req.Endpoint(),
			}

			// Verify the caller has access to the resource
			err = a.Verify(account, res)
			if err != nil && len(account.ID) > 0 {
				return errors.Forbidden(req.Service(), "Forbidden call made to %v:%v by %v", req.Service(), req.Endpoint(), account.ID)
			} else if err != nil {
				return errors.Unauthorized(req.Service(), "Unauthorised call made to %v:%v", req.Service(), req.Endpoint())
			}

			// There is an account, set it in the context
			ctx = auth.ContextWithAccount(ctx, account)

			// The user is authorised, allow the call
			return h(ctx, req, rsp)
		}
	}
}

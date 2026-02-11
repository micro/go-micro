package auth

import (
	"context"

	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/metadata"
)

// ClientOptions for configuring the auth client wrapper
type ClientOptions struct {
	// Auth provider for token generation
	Auth auth.Auth
	// Token to use (optional - if not provided, will be extracted from context)
	Token string
}

// AuthClient returns a client Wrapper that adds authentication tokens to outgoing requests.
//
// For each outgoing request:
// 1. Extracts or uses provided token
// 2. Adds Bearer token to request metadata
// 3. Makes the RPC call
//
// Example usage:
//
//	client := client.NewClient(
//	    client.Wrap(auth.AuthClient(auth.ClientOptions{
//	        Auth:  myAuthProvider,
//	        Token: myToken,
//	    })),
//	)
func AuthClient(opts ClientOptions) client.Wrapper {
	return func(c client.Client) client.Client {
		return &authClient{
			Client: c,
			opts:   opts,
		}
	}
}

// authClient wraps a client to add authentication
type authClient struct {
	client.Client
	opts ClientOptions
}

// Call adds authentication token to the request
func (a *authClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	// Get token from options or context
	token := a.opts.Token
	if token == "" && a.opts.Auth != nil {
		// Try to get token from context account
		if acc, ok := auth.AccountFromContext(ctx); ok {
			// Generate token for this account
			if t, err := a.opts.Auth.Token(auth.WithCredentials(acc.ID, acc.Secret)); err == nil {
				token = t.AccessToken
			}
		}
	}

	// Add token to metadata if available
	if token != "" {
		md, ok := metadata.FromContext(ctx)
		if !ok {
			md = metadata.Metadata{}
		}
		md = TokenToMetadata(md, token)
		ctx = metadata.NewContext(ctx, md)
	}

	return a.Client.Call(ctx, req, rsp, opts...)
}

// Stream adds authentication token to the stream request
func (a *authClient) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	// Get token from options or context
	token := a.opts.Token
	if token == "" && a.opts.Auth != nil {
		// Try to get token from context account
		if acc, ok := auth.AccountFromContext(ctx); ok {
			// Generate token for this account
			if t, err := a.opts.Auth.Token(auth.WithCredentials(acc.ID, acc.Secret)); err == nil {
				token = t.AccessToken
			}
		}
	}

	// Add token to metadata if available
	if token != "" {
		md, ok := metadata.FromContext(ctx)
		if !ok {
			md = metadata.Metadata{}
		}
		md = TokenToMetadata(md, token)
		ctx = metadata.NewContext(ctx, md)
	}

	return a.Client.Stream(ctx, req, opts...)
}

// Publish adds authentication token to the publish request
func (a *authClient) Publish(ctx context.Context, msg client.Message, opts ...client.PublishOption) error {
	// Get token from options or context
	token := a.opts.Token
	if token == "" && a.opts.Auth != nil {
		// Try to get token from context account
		if acc, ok := auth.AccountFromContext(ctx); ok {
			// Generate token for this account
			if t, err := a.opts.Auth.Token(auth.WithCredentials(acc.ID, acc.Secret)); err == nil {
				token = t.AccessToken
			}
		}
	}

	// Add token to metadata if available
	if token != "" {
		md, ok := metadata.FromContext(ctx)
		if !ok {
			md = metadata.Metadata{}
		}
		md = TokenToMetadata(md, token)
		ctx = metadata.NewContext(ctx, md)
	}

	return a.Client.Publish(ctx, msg, opts...)
}

// FromToken creates a client wrapper with a static token.
// This is useful when you have a pre-generated token and don't need the auth provider.
func FromToken(token string) client.Wrapper {
	return AuthClient(ClientOptions{
		Token: token,
	})
}

// FromContext creates a client wrapper that extracts the account from context
// and generates a token for each request. Useful for service-to-service auth.
func FromContext(authProvider auth.Auth) client.Wrapper {
	return AuthClient(ClientOptions{
		Auth: authProvider,
	})
}

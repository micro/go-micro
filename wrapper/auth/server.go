package auth

import (
	"context"

	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/errors"
	"go-micro.dev/v5/metadata"
	"go-micro.dev/v5/server"
)

// HandlerOptions for configuring the auth handler wrapper
type HandlerOptions struct {
	// Auth provider for token verification
	Auth auth.Auth
	// Rules for authorization checks
	Rules auth.Rules
	// SkipEndpoints is a list of endpoints that don't require auth
	// Format: "Service.Method" e.g., "Greeter.Hello"
	SkipEndpoints []string
}

// AuthHandler returns a server HandlerWrapper that enforces authentication and authorization.
//
// For each incoming request:
// 1. Extracts Bearer token from metadata
// 2. Verifies token using auth.Inspect()
// 3. Checks authorization using rules.Verify()
// 4. Adds account to context
// 5. Calls the handler if authorized
//
// Returns 401 Unauthorized if token is missing/invalid.
// Returns 403 Forbidden if account lacks necessary permissions.
//
// Example usage:
//
//	service := micro.NewService(
//	    micro.WrapHandler(auth.AuthHandler(auth.HandlerOptions{
//	        Auth:  myAuthProvider,
//	        Rules: myRules,
//	        SkipEndpoints: []string{"Health.Check"},
//	    })),
//	)
func AuthHandler(opts HandlerOptions) server.HandlerWrapper {
	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			// Get endpoint name
			endpoint := req.Endpoint()

			// Check if this endpoint should skip auth
			for _, skip := range opts.SkipEndpoints {
				if skip == endpoint {
					// Skip auth, proceed to handler
					return h(ctx, req, rsp)
				}
			}

			// Extract metadata from context
			md, ok := metadata.FromContext(ctx)
			if !ok {
				return errors.Unauthorized(req.Service(), "missing metadata")
			}

			// Extract and verify token
			token, err := TokenFromMetadata(md)
			if err != nil {
				if err == ErrMissingToken {
					return errors.Unauthorized(req.Service(), "missing authorization token")
				}
				return errors.Unauthorized(req.Service(), "invalid authorization token: %v", err)
			}

			// Verify token and get account
			var account *auth.Account
			if opts.Auth != nil {
				account, err = opts.Auth.Inspect(token)
				if err != nil {
					if err == auth.ErrInvalidToken {
						return errors.Unauthorized(req.Service(), "invalid token")
					}
					return errors.Unauthorized(req.Service(), "token verification failed: %v", err)
				}
			}

			// Check authorization if rules are provided
			if opts.Rules != nil && account != nil {
				resource := &auth.Resource{
					Name:     req.Service(),
					Type:     "service",
					Endpoint: endpoint,
				}

				if err := opts.Rules.Verify(account, resource); err != nil {
					if err == auth.ErrForbidden {
						return errors.Forbidden(req.Service(), "access denied to %s", endpoint)
					}
					return errors.Forbidden(req.Service(), "authorization failed: %v", err)
				}
			}

			// Add account to context for handler to use
			if account != nil {
				ctx = auth.ContextWithAccount(ctx, account)
			}

			// Call the handler
			return h(ctx, req, rsp)
		}
	}
}

// PublicEndpoints is a helper to create auth options that allow public access to specific endpoints.
func PublicEndpoints(authProvider auth.Auth, rules auth.Rules, publicEndpoints []string) HandlerOptions {
	return HandlerOptions{
		Auth:          authProvider,
		Rules:         rules,
		SkipEndpoints: publicEndpoints,
	}
}

// AuthRequired creates auth options that require authentication for all endpoints.
func AuthRequired(authProvider auth.Auth, rules auth.Rules) HandlerOptions {
	return HandlerOptions{
		Auth:          authProvider,
		Rules:         rules,
		SkipEndpoints: []string{},
	}
}

// AuthOptional creates auth options that extract auth if present but don't enforce it.
// Useful for endpoints that behave differently for authenticated users but also work without auth.
func AuthOptional(authProvider auth.Auth) server.HandlerWrapper {
	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			// Try to extract account, but don't fail if missing
			md, ok := metadata.FromContext(ctx)
			if ok {
				if token, err := TokenFromMetadata(md); err == nil {
					if account, err := authProvider.Inspect(token); err == nil {
						ctx = auth.ContextWithAccount(ctx, account)
					}
				}
			}

			// Always call handler, with or without account in context
			return h(ctx, req, rsp)
		}
	}
}

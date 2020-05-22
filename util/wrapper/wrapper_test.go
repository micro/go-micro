package wrapper

import (
	"context"
	"net/http"
	"testing"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/server"
)

func TestWrapper(t *testing.T) {
	testData := []struct {
		existing  metadata.Metadata
		headers   metadata.Metadata
		overwrite bool
	}{
		{
			existing: metadata.Metadata{},
			headers: metadata.Metadata{
				"Foo": "bar",
			},
			overwrite: true,
		},
		{
			existing: metadata.Metadata{
				"Foo": "bar",
			},
			headers: metadata.Metadata{
				"Foo": "baz",
			},
			overwrite: false,
		},
	}

	for _, d := range testData {
		c := &fromServiceWrapper{
			headers: d.headers,
		}

		ctx := metadata.NewContext(context.Background(), d.existing)
		ctx = c.setHeaders(ctx)
		md, _ := metadata.FromContext(ctx)

		for k, v := range d.headers {
			if d.overwrite && md[k] != v {
				t.Fatalf("Expected %s=%s got %s=%s", k, v, k, md[k])
			}
			if !d.overwrite && md[k] != d.existing[k] {
				t.Fatalf("Expected %s=%s got %s=%s", k, d.existing[k], k, md[k])
			}
		}
	}
}

type testAuth struct {
	verifyCount    int
	inspectCount   int
	namespace      string
	inspectAccount *auth.Account
	verifyError    error

	auth.Auth
}

func (a *testAuth) Verify(acc *auth.Account, res *auth.Resource, opts ...auth.VerifyOption) error {
	a.verifyCount = a.verifyCount + 1
	return a.verifyError
}

func (a *testAuth) Inspect(token string) (*auth.Account, error) {
	a.inspectCount = a.inspectCount + 1
	return a.inspectAccount, nil
}

func (a *testAuth) Options() auth.Options {
	return auth.Options{Namespace: a.namespace}
}

type testRequest struct {
	service  string
	endpoint string

	server.Request
}

func (r testRequest) Service() string {
	return r.service
}

func (r testRequest) Endpoint() string {
	return r.endpoint
}

func TestAuthHandler(t *testing.T) {
	h := func(ctx context.Context, req server.Request, rsp interface{}) error {
		return nil
	}

	debugReq := testRequest{service: "go.micro.service.foo", endpoint: "Debug.Foo"}
	serviceReq := testRequest{service: "go.micro.service.foo", endpoint: "Foo.Bar"}

	// Debug endpoints should be excluded from auth so auth.Verify should never get called
	t.Run("DebugEndpoint", func(t *testing.T) {
		a := testAuth{}
		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		err := handler(h)(context.TODO(), debugReq, nil)
		if err != nil {
			t.Errorf("Expected nil error but got %v", err)
		}
		if a.verifyCount != 0 {
			t.Errorf("Did not expect verify to be called")
		}
	})

	// If the Authorization header is blank, no error should be returned and verify not called
	t.Run("BlankAuthorizationHeader", func(t *testing.T) {
		a := testAuth{}
		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		err := handler(h)(context.TODO(), serviceReq, nil)
		if err != nil {
			t.Errorf("Expected nil error but got %v", err)
		}
		if a.inspectCount != 0 {
			t.Errorf("Did not expect inspect to be called")
		}
	})

	// If the Authorization header is invalid, an error should be returned and verify not called
	t.Run("InvalidAuthorizationHeader", func(t *testing.T) {
		a := testAuth{}
		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		ctx := metadata.Set(context.TODO(), "Authorization", "Invalid")
		err := handler(h)(ctx, serviceReq, nil)
		if verr, ok := err.(*errors.Error); !ok || verr.Code != http.StatusUnauthorized {
			t.Errorf("Expected unauthorized error but got %v", err)
		}
		if a.inspectCount != 0 {
			t.Errorf("Did not expect inspect to be called")
		}
	})

	// If the Authorization header is valid, no error should be returned and verify should called
	t.Run("ValidAuthorizationHeader", func(t *testing.T) {
		a := testAuth{}
		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		ctx := metadata.Set(context.TODO(), "Authorization", auth.BearerScheme+"Token")
		err := handler(h)(ctx, serviceReq, nil)
		if err != nil {
			t.Errorf("Expected nil error but got %v", err)
		}
		if a.inspectCount != 1 {
			t.Errorf("Expected inspect to be called")
		}
	})

	// If the namespace header was not set on the request, the wrapper should set it to the auths
	// own namespace
	t.Run("BlankNamespaceHeader", func(t *testing.T) {
		a := testAuth{namespace: "mynamespace"}
		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		inCtx := context.TODO()
		h := func(ctx context.Context, req server.Request, rsp interface{}) error {
			inCtx = ctx
			return nil
		}

		err := handler(h)(inCtx, serviceReq, nil)
		if err != nil {
			t.Errorf("Expected nil error but got %v", err)
		}
		if ns, _ := metadata.Get(inCtx, "Micro-Namespace"); ns != a.namespace {
			t.Errorf("Expected namespace to be set to %v but was %v", a.namespace, ns)
		}
	})
	t.Run("ValidNamespaceHeader", func(t *testing.T) {
		a := testAuth{namespace: "mynamespace"}
		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		inNs := "reqnamespace"
		inCtx := metadata.Set(context.TODO(), "Micro-Namespace", inNs)
		h := func(ctx context.Context, req server.Request, rsp interface{}) error {
			inCtx = ctx
			return nil
		}

		err := handler(h)(inCtx, serviceReq, nil)
		if err != nil {
			t.Errorf("Expected nil error but got %v", err)
		}
		if ns, _ := metadata.Get(inCtx, "Micro-Namespace"); ns != inNs {
			t.Errorf("Expected namespace to remain as %v but was set to %v", inNs, ns)
		}
	})

	// If the callers account was set but the issuer didn't match that of the request, the request
	// should be forbidden
	t.Run("InvalidAccountIssuer", func(t *testing.T) {
		a := testAuth{
			namespace:      "validnamespace",
			inspectAccount: &auth.Account{Issuer: "invalidnamespace"},
		}

		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		ctx := metadata.Set(context.TODO(), "Authorization", auth.BearerScheme+"Token")
		err := handler(h)(ctx, serviceReq, nil)
		if verr, ok := err.(*errors.Error); !ok || verr.Code != http.StatusForbidden {
			t.Errorf("Expected forbidden error but got %v", err)
		}
	})
	t.Run("ValidAccountIssuer", func(t *testing.T) {
		a := testAuth{
			namespace:      "validnamespace",
			inspectAccount: &auth.Account{Issuer: "validnamespace"},
		}

		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		ctx := metadata.Set(context.TODO(), "Authorization", auth.BearerScheme+"Token")
		err := handler(h)(ctx, serviceReq, nil)
		if err != nil {
			t.Errorf("Expected nil error but got %v", err)
		}
	})

	// If the caller had a nil account and verify returns an error, the request should be unauthorised
	t.Run("NilAccountUnauthorized", func(t *testing.T) {
		a := testAuth{verifyError: auth.ErrForbidden}

		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		err := handler(h)(context.TODO(), serviceReq, nil)
		if verr, ok := err.(*errors.Error); !ok || verr.Code != http.StatusUnauthorized {
			t.Errorf("Expected unauthorizard error but got %v", err)
		}
	})
	t.Run("AccountForbidden", func(t *testing.T) {
		a := testAuth{verifyError: auth.ErrForbidden, inspectAccount: &auth.Account{}}

		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		ctx := metadata.Set(context.TODO(), "Authorization", auth.BearerScheme+"Token")
		err := handler(h)(ctx, serviceReq, nil)
		if verr, ok := err.(*errors.Error); !ok || verr.Code != http.StatusForbidden {
			t.Errorf("Expected forbidden error but got %v", err)
		}
	})
	t.Run("AccountValid", func(t *testing.T) {
		a := testAuth{inspectAccount: &auth.Account{}}

		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		ctx := metadata.Set(context.TODO(), "Authorization", auth.BearerScheme+"Token")
		err := handler(h)(ctx, serviceReq, nil)
		if err != nil {
			t.Errorf("Expected nil error but got %v", err)
		}
	})

	// If an account is returned from inspecting the token, it should be set in the context
	t.Run("ContextWithAccount", func(t *testing.T) {
		accID := "myaccountid"
		a := testAuth{inspectAccount: &auth.Account{ID: accID}}

		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		inCtx := metadata.Set(context.TODO(), "Authorization", auth.BearerScheme+"Token")
		h := func(ctx context.Context, req server.Request, rsp interface{}) error {
			inCtx = ctx
			return nil
		}

		err := handler(h)(inCtx, serviceReq, nil)
		if err != nil {
			t.Errorf("Expected nil error but got %v", err)
		}
		if acc, ok := auth.AccountFromContext(inCtx); !ok {
			t.Errorf("Expected an account to be set in the context")
		} else if acc.ID != accID {
			t.Errorf("Expected the account in the context to have the ID %v but it actually had %v", accID, acc.ID)
		}
	})

	// If verify returns an error the handler should not be called
	t.Run("HandlerNotCalled", func(t *testing.T) {
		a := testAuth{verifyError: auth.ErrForbidden}

		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		var handlerCalled bool
		h := func(ctx context.Context, req server.Request, rsp interface{}) error {
			handlerCalled = true
			return nil
		}

		ctx := metadata.Set(context.TODO(), "Authorization", auth.BearerScheme+"Token")
		err := handler(h)(ctx, serviceReq, nil)
		if verr, ok := err.(*errors.Error); !ok || verr.Code != http.StatusUnauthorized {
			t.Errorf("Expected unauthorizard error but got %v", err)
		}
		if handlerCalled {
			t.Errorf("Expected the handler to not be called")
		}
	})

	// If verify does not return an error the handler should be called
	t.Run("HandlerNotCalled", func(t *testing.T) {
		a := testAuth{}

		handler := AuthHandler(func() auth.Auth {
			return &a
		})

		var handlerCalled bool
		h := func(ctx context.Context, req server.Request, rsp interface{}) error {
			handlerCalled = true
			return nil
		}

		ctx := metadata.Set(context.TODO(), "Authorization", auth.BearerScheme+"Token")
		err := handler(h)(ctx, serviceReq, nil)
		if err != nil {
			t.Errorf("Expected nil error but got %v", err)
		}
		if !handlerCalled {
			t.Errorf("Expected the handler be called")
		}
	})
}

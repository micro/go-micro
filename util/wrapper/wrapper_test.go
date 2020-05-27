package wrapper

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/client"
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

type testClient struct {
	callCount int
	callRsp   interface{}
	client.Client
}

func (c *testClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	c.callCount++

	if c.callRsp != nil {
		val := reflect.ValueOf(rsp).Elem()
		val.Set(reflect.ValueOf(c.callRsp).Elem())
	}

	return nil
}

type testRsp struct {
	value string
}

func TestCacheWrapper(t *testing.T) {
	req := client.NewRequest("go.micro.service.foo", "Foo.Bar", nil)

	t.Run("NilCache", func(t *testing.T) {
		cli := new(testClient)

		w := CacheClient(func() *client.Cache {
			return nil
		}, cli)

		// perfroming two requests should increment the call count by two indicating the cache wasn't
		// used even though the WithCache option was passed.
		w.Call(context.TODO(), req, nil, client.WithCache(time.Minute))
		w.Call(context.TODO(), req, nil, client.WithCache(time.Minute))

		if cli.callCount != 2 {
			t.Errorf("Expected the client to have been called twice")
		}
	})

	t.Run("OptionNotSet", func(t *testing.T) {
		cli := new(testClient)
		cache := client.NewCache()

		w := CacheClient(func() *client.Cache {
			return cache
		}, cli)

		// perfroming two requests should increment the call count by two since we didn't pass the WithCache
		// option to Call.
		w.Call(context.TODO(), req, nil)
		w.Call(context.TODO(), req, nil)

		if cli.callCount != 2 {
			t.Errorf("Expected the client to have been called twice")
		}
	})

	t.Run("OptionSet", func(t *testing.T) {
		val := "foo"
		cli := &testClient{callRsp: &testRsp{value: val}}
		cache := client.NewCache()

		w := CacheClient(func() *client.Cache {
			return cache
		}, cli)

		// perfroming two requests should increment the call count by once since the second request should
		// have used the cache. The correct value should be set on both responses and no errors should
		// be returned.
		rsp1 := &testRsp{}
		rsp2 := &testRsp{}
		err1 := w.Call(context.TODO(), req, rsp1, client.WithCache(time.Minute))
		err2 := w.Call(context.TODO(), req, rsp2, client.WithCache(time.Minute))

		if err1 != nil {
			t.Errorf("Expected nil error, got %v", err1)
		}
		if err2 != nil {
			t.Errorf("Expected nil error, got %v", err2)
		}

		if rsp1.value != val {
			t.Errorf("Expected %v to be assigned to the value, got %v", val, rsp1.value)
		}
		if rsp2.value != val {
			t.Errorf("Expected %v to be assigned to the value, got %v", val, rsp2.value)
		}

		if cli.callCount != 1 {
			t.Errorf("Expected the client to be called 1 time, was actually called %v time(s)", cli.callCount)
		}
	})
}

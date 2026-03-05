package wrapper

import (
	"context"
	"reflect"
	"testing"

	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/metadata"
	"go-micro.dev/v5/server"
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

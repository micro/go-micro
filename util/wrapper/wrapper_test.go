package wrapper

import (
	"context"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/metadata"
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

type testClient struct {
	callCount int
	callRsp   interface{}
	cache     *client.Cache
	client.Client
}

func (c *testClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	c.callCount++
	rsp = c.callRsp
	return nil
}

func (c *testClient) Options() client.Options {
	return client.Options{Cache: c.cache}
}
func TestCacheWrapper(t *testing.T) {
	req := client.NewRequest("go.micro.service.foo", "Foo.Bar", nil)

	t.Run("NilCache", func(t *testing.T) {
		cli := new(testClient)
		w := CacheClient(cli)

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
		w := CacheClient(cli)

		// perfroming two requests should increment the call count by two since we didn't pass the WithCache
		// option to Call.
		w.Call(context.TODO(), req, nil)
		w.Call(context.TODO(), req, nil)

		if cli.callCount != 2 {
			t.Errorf("Expected the client to have been called twice")
		}
	})

	t.Run("OptionSet", func(t *testing.T) {
		cli := &testClient{callRsp: "foobar", cache: client.NewCache()}
		w := CacheClient(cli)

		// perfroming two requests should increment the call count by once since the second request should
		// have used the cache
		err1 := w.Call(context.TODO(), req, nil, client.WithCache(time.Minute))
		err2 := w.Call(context.TODO(), req, nil, client.WithCache(time.Minute))

		if err1 != nil {
			t.Errorf("Expected nil error, got %v", err1)
		}
		if err2 != nil {
			t.Errorf("Expected nil error, got %v", err2)
		}
		if cli.callCount != 1 {
			t.Errorf("Expected the client to be called 1 time, was actually called %v time(s)", cli.callCount)
		}
	})
}

package wrapper

import (
	"context"
	"reflect"
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

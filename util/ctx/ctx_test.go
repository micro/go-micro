package ctx

import (
	"net/http"
	"testing"

	"github.com/micro/go-micro/metadata"
)

func TestRequestToContext(t *testing.T) {
	testData := []struct {
		request *http.Request
		expect  metadata.Metadata
	}{
		{
			&http.Request{
				Header: http.Header{
					"foo1": []string{"bar"},
					"foo2": []string{"bar", "baz"},
				},
			},
			metadata.Metadata{
				"foo1": "bar",
				"foo2": "bar,baz",
			},
		},
	}

	for _, d := range testData {
		ctx := FromRequest(d.request)
		md, ok := metadata.FromContext(ctx)
		if !ok {
			t.Fatalf("Expected metadata for request %+v", d.request)
		}
		for k, v := range d.expect {
			if val := md[k]; val != v {
				t.Fatalf("Expected %s for key %s for expected md %+v, got md %+v", v, k, d.expect, md)
			}
		}
	}
}

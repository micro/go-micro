package api

import (
	"net/http"
	"net/url"
	"testing"
)

func TestRequestToProto(t *testing.T) {
	testData := []*http.Request{
		{
			Method: "GET",
			Header: http.Header{
				"Header": []string{"test"},
			},
			URL: &url.URL{
				Scheme:   "http",
				Host:     "localhost",
				Path:     "/foo/bar",
				RawQuery: "param1=value1",
			},
		},
	}

	for _, d := range testData {
		p, err := requestToProto(d)
		if err != nil {
			t.Fatal(err)
		}
		if p.Path != d.URL.Path {
			t.Fatalf("Expected path %s got %s", d.URL.Path, p.Path)
		}
		if p.Method != d.Method {
			t.Fatalf("Expected method %s got %s", d.Method, p.Method)
		}
		for k, v := range d.Header {
			if val, ok := p.Header[k]; !ok {
				t.Fatalf("Expected header %s", k)
			} else {
				if val.Values[0] != v[0] {
					t.Fatalf("Expected val %s, got %s", val.Values[0], v[0])
				}
			}
		}
	}
}

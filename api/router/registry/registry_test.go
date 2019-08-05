package registry

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/micro/go-micro/api"
)

func TestSetNamespace(t *testing.T) {
	testCases := []struct {
		namespace string
		name      string
		expected  string
	}{
		// default dotted path
		{
			"go.micro.api",
			"foo",
			"go.micro.api.foo",
		},
		// dotted end
		{
			"go.micro.api.",
			"foo",
			"go.micro.api.foo",
		},
		// dashed end
		{
			"go-micro-api-",
			"foo",
			"go-micro-api-foo",
		},
		// no namespace
		{
			"",
			"foo",
			"foo",
		},
		{
			"go-micro-api-",
			"v2.foo",
			"go-micro-api-v2-foo",
		},
	}

	for _, test := range testCases {
		name := setNamespace(test.namespace, test.name)
		if name != test.expected {
			t.Fatalf("expected name %s got %s", test.expected, name)
		}
	}
}

func TestRouter(t *testing.T) {
	r := newRouter()

	compare := func(expect, got []string) bool {
		// no data to compare, return true
		if len(expect) == 0 && len(got) == 0 {
			return true
		}
		// no data expected but got some return false
		if len(expect) == 0 && len(got) > 0 {
			return false
		}

		// compare expected with what we got
		for _, e := range expect {
			var seen bool
			for _, g := range got {
				if e == g {
					seen = true
					break
				}
			}
			if !seen {
				return false
			}
		}

		// we're done, return true
		return true
	}

	testData := []struct {
		e *api.Endpoint
		r *http.Request
		m bool
	}{
		{
			e: &api.Endpoint{
				Name:   "Foo.Bar",
				Host:   []string{"example.com"},
				Method: []string{"GET"},
				Path:   []string{"/foo"},
			},
			r: &http.Request{
				Host:   "example.com",
				Method: "GET",
				URL: &url.URL{
					Path: "/foo",
				},
			},
			m: true,
		},
		{
			e: &api.Endpoint{
				Name:   "Bar.Baz",
				Host:   []string{"example.com", "foo.com"},
				Method: []string{"GET", "POST"},
				Path:   []string{"/foo/bar"},
			},
			r: &http.Request{
				Host:   "foo.com",
				Method: "POST",
				URL: &url.URL{
					Path: "/foo/bar",
				},
			},
			m: true,
		},
		{
			e: &api.Endpoint{
				Name:   "Test.Cruft",
				Host:   []string{"example.com", "foo.com"},
				Method: []string{"GET", "POST"},
				Path:   []string{"/xyz"},
			},
			r: &http.Request{
				Host:   "fail.com",
				Method: "DELETE",
				URL: &url.URL{
					Path: "/test/fail",
				},
			},
			m: false,
		},
	}

	for _, d := range testData {
		key := fmt.Sprintf("%s:%s", "test.service", d.e.Name)
		r.eps[key] = &api.Service{
			Endpoint: d.e,
		}
	}

	for _, d := range testData {
		e, err := r.Endpoint(d.r)
		if d.m && err != nil {
			t.Fatalf("expected match, got %v", err)
		}
		if !d.m && err == nil {
			t.Fatal("expected error got match")
		}
		// skip testing the non match
		if !d.m {
			continue
		}

		ep := e.Endpoint

		// test the match
		if d.e.Name != ep.Name {
			t.Fatalf("expected %v got %v", d.e.Name, ep.Name)
		}
		if ok := compare(d.e.Method, ep.Method); !ok {
			t.Fatalf("expected %v got %v", d.e.Method, ep.Method)
		}
		if ok := compare(d.e.Path, ep.Path); !ok {
			t.Fatalf("expected %v got %v", d.e.Path, ep.Path)
		}
		if ok := compare(d.e.Host, ep.Host); !ok {
			t.Fatalf("expected %v got %v", d.e.Host, ep.Host)
		}

	}

}

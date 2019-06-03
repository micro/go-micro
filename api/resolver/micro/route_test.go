package micro

import (
	"testing"
)

func TestApiRoute(t *testing.T) {
	testData := []struct {
		path    string
		service string
		method  string
	}{
		{
			"/foo/bar",
			"foo",
			"Foo.Bar",
		},
		{
			"/foo/foo/bar",
			"foo",
			"Foo.Bar",
		},
		{
			"/foo/bar/baz",
			"foo",
			"Bar.Baz",
		},
		{
			"/foo/bar/baz-xyz",
			"foo",
			"Bar.BazXyz",
		},
		{
			"/foo/bar/baz/cat",
			"foo.bar",
			"Baz.Cat",
		},
		{
			"/foo/bar/baz/cat/car",
			"foo.bar.baz",
			"Cat.Car",
		},
		{
			"/foo/fooBar/bazCat",
			"foo",
			"FooBar.BazCat",
		},
		{
			"/v1/foo/bar",
			"v1.foo",
			"Foo.Bar",
		},
		{
			"/v1/foo/bar/baz",
			"v1.foo",
			"Bar.Baz",
		},
		{
			"/v1/foo/bar/baz/cat",
			"v1.foo.bar",
			"Baz.Cat",
		},
	}

	for _, d := range testData {
		s, m := apiRoute(d.path)
		if d.service != s {
			t.Fatalf("Expected service: %s for path: %s got: %s %s", d.service, d.path, s, m)
		}
		if d.method != m {
			t.Fatalf("Expected service: %s for path: %s got: %s", d.method, d.path, m)
		}
	}
}

func TestProxyRoute(t *testing.T) {
	testData := []struct {
		path    string
		service string
	}{
		// no namespace
		{
			"/f",
			"f",
		},
		{
			"/f",
			"f",
		},
		{
			"/f-b",
			"f-b",
		},
		{
			"/foo/bar",
			"foo",
		},
		{
			"/foo-bar",
			"foo-bar",
		},
		{
			"/foo-bar-baz",
			"foo-bar-baz",
		},
		{
			"/foo/bar/bar",
			"foo",
		},
		{
			"/v1/foo/bar",
			"v1.foo",
		},
		{
			"/v1/foo/bar/baz",
			"v1.foo",
		},
		{
			"/v1/foo/bar/baz/cat",
			"v1.foo",
		},
	}

	for _, d := range testData {
		s := proxyRoute(d.path)
		if d.service != s {
			t.Fatalf("Expected service: %s for path: %s got: %s", d.service, d.path, s)
		}
	}
}

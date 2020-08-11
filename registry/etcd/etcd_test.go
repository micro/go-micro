package etcd

import (
	"testing"
)

// test whether the name matches
func TestEtcdHasName(t *testing.T) {
	testCases := []struct {
		key    string
		prefix string
		name   string
		domain string
		expect bool
	}{
		{
			"/micro/registry/micro/registry",
			"/micro/registry",
			"registry",
			"micro",
			true,
		},
		{
			"/micro/registry/micro",
			"/micro/registry",
			"store",
			"micro",
			false,
		},
		{
			"/prefix/baz/*/registry",
			"/prefix/baz",
			"registry",
			"*",
			true,
		},
		{
			"/prefix/baz",
			"/prefix/baz",
			"store",
			"micro",
			false,
		},
		{
			"/prefix/baz/foobar/registry",
			"/prefix/baz",
			"registry",
			"foobar",
			true,
		},
	}

	for _, c := range testCases {
		domain, service, ok := getName(c.key, c.prefix)
		if ok != c.expect {
			t.Fatalf("Expected %t for %v got: %t", c.expect, c, ok)
		}
		if !ok {
			continue
		}
		if service != c.name {
			t.Fatalf("Expected service %s got %s", c.name, service)
		}
		if domain != c.domain {
			t.Fatalf("Expected domain %s got %s", c.domain, domain)
		}
	}
}

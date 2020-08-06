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
		expect bool
	}{
		{
			"/micro/registry/micro/registry",
			"/micro/registry",
			"registry",
			true,
		},
		{
			"/micro/registry/micro/store",
			"/micro/registry",
			"registry",
			false,
		},
		{
			"/prefix/baz/*/registry",
			"/prefix/baz",
			"registry",
			true,
		},
		{
			"/prefix/baz/micro/registry",
			"/prefix/baz",
			"store",
			false,
		},
		{
			"/prefix/baz/micro/registry",
			"/prefix/baz",
			"registry",
			true,
		},
	}

	for _, c := range testCases {
		v := hasName(c.key, c.prefix, c.name)
		if v != c.expect {
			t.Fatalf("Expected %t for %v got: %t", c.expect, c, v)
		}
	}
}

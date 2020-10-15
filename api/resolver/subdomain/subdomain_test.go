package subdomain

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/micro/go-micro/v3/api/resolver/vpath"

	"github.com/stretchr/testify/assert"
)

func TestResolve(t *testing.T) {
	tt := []struct {
		Name   string
		Host   string
		Result string
	}{
		{
			Name:   "Top level domain",
			Host:   "micro.mu",
			Result: "micro",
		},
		{
			Name:   "Effective top level domain",
			Host:   "micro.com.au",
			Result: "micro",
		},
		{
			Name:   "Subdomain dev",
			Host:   "dev.micro.mu",
			Result: "dev",
		},
		{
			Name:   "Subdomain foo",
			Host:   "foo.micro.mu",
			Result: "foo",
		},
		{
			Name:   "Multi-level subdomain",
			Host:   "staging.myapp.m3o.app",
			Result: "myapp-staging",
		},
		{
			Name:   "Dev host",
			Host:   "127.0.0.1",
			Result: "micro",
		},
		{
			Name:   "Localhost",
			Host:   "localhost",
			Result: "micro",
		},
		{
			Name:   "IP host",
			Host:   "81.151.101.146",
			Result: "micro",
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			r := NewResolver(vpath.NewResolver())
			result, err := r.Resolve(&http.Request{URL: &url.URL{Host: tc.Host, Path: "foo/bar"}})
			assert.Nil(t, err, "Expecter err to be nil")
			if result != nil {
				assert.Equal(t, tc.Result, result.Domain, "Expected %v but got %v", tc.Result, result.Domain)
			}
		})
	}
}

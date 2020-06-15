package registry

import (
	"testing"

	"github.com/micro/go-micro/v2/registry"
	"github.com/stretchr/testify/assert"
)

func TestStoreRegex(t *testing.T) {
	router := newRouter()
	router.store([]*registry.Service{
		{
			Name:    "Foobar",
			Version: "latest",
			Endpoints: []*registry.Endpoint{
				{
					Name: "foo",
					Metadata: map[string]string{
						"endpoint":    "FooEndpoint",
						"description": "Some description",
						"method":      "POST",
						"path":        "^/foo/$",
						"handler":     "rpc",
					},
				},
			},
			Metadata: map[string]string{},
		},
	},
	)

	assert.Len(t, router.ceps["Foobar.foo"].pcreregs, 1)
}

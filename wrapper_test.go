package micro

import (
	"context"
	"testing"

	"github.com/micro/go-micro/metadata"
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
				"foo": "bar",
			},
			overwrite: true,
		},
		{
			existing: metadata.Metadata{
				"foo": "bar",
			},
			headers: metadata.Metadata{
				"foo": "baz",
			},
			overwrite: false,
		},
	}

	for _, d := range testData {
		c := &clientWrapper{
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

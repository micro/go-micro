package mdns

import (
	"testing"

	"github.com/micro/go-micro/registry"
)

func TestEncoding(t *testing.T) {
	testData := []*mdnsTxt{
		&mdnsTxt{
			Version: "1.0.0",
			Metadata: map[string]string{
				"foo": "bar",
			},
			Endpoints: []*registry.Endpoint{
				&registry.Endpoint{
					Name: "endpoint1",
					Request: &registry.Value{
						Name: "request",
						Type: "request",
					},
					Response: &registry.Value{
						Name: "response",
						Type: "response",
					},
					Metadata: map[string]string{
						"foo1": "bar1",
					},
				},
			},
		},
	}

	for _, d := range testData {
		encoded, err := encode(d)
		if err != nil {
			t.Fatal(err)
		}

		for _, txt := range encoded {
			if len(txt) > 255 {
				t.Fatalf("One of parts for txt is %d characters", len(txt))
			}
		}

		decoded, err := decode(encoded)
		if err != nil {
			t.Fatal(err)
		}

		if decoded.Version != d.Version {
			t.Fatalf("Expected version %s got %s", d.Version, decoded.Version)
		}

		if len(decoded.Endpoints) != len(d.Endpoints) {
			t.Fatalf("Expected %d endpoints, got %d", len(d.Endpoints), len(decoded.Endpoints))
		}

		for k, v := range d.Metadata {
			if val := decoded.Metadata[k]; val != v {
				t.Fatalf("Expected %s=%s got %s=%s", k, v, k, val)
			}
		}
	}

}

package registry

import (
	"testing"
)

func TestEncodingEndpoints(t *testing.T) {
	testData := []struct {
		decoded    *Endpoint
		encoded    string
		oldEncoded string
	}{
		{
			&Endpoint{
				Name: "Endpoint",
				Request: &Value{
					Name: "request",
					Type: "Request",
				},
				Response: &Value{
					Name: "response",
					Type: "Response",
				},
				Metadata: map[string]string{
					"foo": "bar",
				},
			},
			"e-789caa56ca4bcc4d55b25272cd4b29c8cfcc2b51d2512a4a2d2c4d2d2e51b2824bc24474944a2a0b4002417081b2c41c204bc92aaf3427a716a4b7b8203faf38154533540849375c044d7b6e6a49624a624922487b5a7e3e506d526291526d2d200000ffffb9fb3937",
			`e={"name":"Endpoint","request":{"name":"request","type":"Request","values":null},"response":{"name":"response","type":"Response","values":null},"metadata":{"foo":"bar"}}`,
		},
	}

	for _, data := range testData {
		e := encodeEndpoints([]*Endpoint{data.decoded})

		if len(e) != 1 || e[0] != data.encoded {
			t.Fatalf("Expected %s got %s", data.encoded, e)
		}

		d := decodeEndpoints(e)
		if len(d) == 0 {
			t.Fatalf("Expected %v got %v", data.decoded, d)
		}

		if d[0].Name != data.decoded.Name {
			t.Fatalf("Expected ep %s got %s", data.decoded.Name, d[0].Name)
		}

		for k, v := range data.decoded.Metadata {
			if gv := d[0].Metadata[k]; gv != v {
				t.Fatalf("Expected key %s val %s got val %s", k, v, gv)
			}
		}

		d = decodeEndpoints([]string{data.oldEncoded})
		if len(d) == 0 {
			t.Fatalf("Expected %v got %v", data.decoded, d)
		}

		if d[0].Name != data.decoded.Name {
			t.Fatalf("Expected ep %s got %s", data.decoded.Name, d[0].Name)
		}

		for k, v := range data.decoded.Metadata {
			if gv := d[0].Metadata[k]; gv != v {
				t.Fatalf("Expected key %s val %s got val %s", k, v, gv)
			}
		}
	}
}

func TestEncodingVersion(t *testing.T) {
	testData := []struct {
		decoded    string
		encoded    string
		oldEncoded string
	}{
		{"1.0.0", "v-789c32d433d03300040000ffff02ce00ee", "v=1.0.0"},
		{"latest", "v-789cca492c492d2e01040000ffff08cc028e", "v=latest"},
	}

	for _, data := range testData {
		e := encodeVersion(data.decoded)

		if e != data.encoded {
			t.Fatalf("Expected %s got %s", data.encoded, e)
		}

		d, ok := decodeVersion([]string{e})
		if !ok {
			t.Fatal("Unexpected %t for %s", ok, data.encoded)
		}

		if d != data.decoded {
			t.Fatal("Expected %s got %s", data.decoded, d)
		}

		d, ok = decodeVersion([]string{data.encoded})
		if !ok {
			t.Fatal("Unexpected %t for %s", ok, data.encoded)
		}

		if d != data.decoded {
			t.Fatal("Expected %s got %s", data.decoded, d)
		}

		d, ok = decodeVersion([]string{data.oldEncoded})
		if !ok {
			t.Fatal("Unexpected %t for %s", ok, data.oldEncoded)
		}

		if d != data.decoded {
			t.Fatal("Expected %s got %s", data.decoded, d)
		}
	}
}

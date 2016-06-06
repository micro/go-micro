package registry

import (
	"encoding/json"

	"testing"
)

func TestEncodingEndpoints(t *testing.T) {
	eps := []*Endpoint{
		&Endpoint{
			Name: "endpoint1",
			Request: &Value{
				Name: "request",
				Type: "request",
			},
			Response: &Value{
				Name: "response",
				Type: "response",
			},
			Metadata: map[string]string{
				"foo1": "bar1",
			},
		},
		&Endpoint{
			Name: "endpoint2",
			Request: &Value{
				Name: "request",
				Type: "request",
			},
			Response: &Value{
				Name: "response",
				Type: "response",
			},
			Metadata: map[string]string{
				"foo2": "bar2",
			},
		},
		&Endpoint{
			Name: "endpoint3",
			Request: &Value{
				Name: "request",
				Type: "request",
			},
			Response: &Value{
				Name: "response",
				Type: "response",
			},
			Metadata: map[string]string{
				"foo3": "bar3",
			},
		},
	}

	testEp := func(ep *Endpoint, enc string) {
		// encode endpoint
		e := encodeEndpoints([]*Endpoint{ep})

		// check there are two tags; old and new
		if len(e) != 1 {
			t.Fatalf("Expected 1 encoded tags, got %v", e)
		}

		// check old encoding
		var seen bool

		for _, en := range e {
			if en == enc {
				seen = true
				break
			}
		}

		if !seen {
			t.Fatalf("Expected %s but not found", enc)
		}

		// decode
		d := decodeEndpoints([]string{enc})
		if len(d) == 0 {
			t.Fatalf("Expected %v got %v", ep, d)
		}

		// check name
		if d[0].Name != ep.Name {
			t.Fatalf("Expected ep %s got %s", ep.Name, d[0].Name)
		}

		// check all the metadata exists
		for k, v := range ep.Metadata {
			if gv := d[0].Metadata[k]; gv != v {
				t.Fatalf("Expected key %s val %s got val %s", k, v, gv)
			}
		}
	}

	for _, ep := range eps {
		// JSON encoded
		jencoded, err := json.Marshal(ep)
		if err != nil {
			t.Fatal(err)
		}

		// HEX encoded
		hencoded := encode(jencoded)
		// endpoint tag
		hepTag := "e-" + hencoded
		testEp(ep, hepTag)
	}
}

func TestEncodingVersion(t *testing.T) {
	testData := []struct {
		decoded string
		encoded string
	}{
		{"1.0.0", "v-789c32d433d03300040000ffff02ce00ee"},
		{"latest", "v-789cca492c492d2e01040000ffff08cc028e"},
	}

	for _, data := range testData {
		e := encodeVersion(data.decoded)

		if e[0] != data.encoded {
			t.Fatalf("Expected %s got %s", data.encoded, e)
		}

		d, ok := decodeVersion(e)
		if !ok {
			t.Fatalf("Unexpected %t for %s", ok, data.encoded)
		}

		if d != data.decoded {
			t.Fatalf("Expected %s got %s", data.decoded, d)
		}

		d, ok = decodeVersion([]string{data.encoded})
		if !ok {
			t.Fatalf("Unexpected %t for %s", ok, data.encoded)
		}

		if d != data.decoded {
			t.Fatalf("Expected %s got %s", data.decoded, d)
		}
	}
}

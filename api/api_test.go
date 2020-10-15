package api

import (
	"strings"
	"testing"
)

func TestEncoding(t *testing.T) {
	testData := []*Endpoint{
		nil,
		{
			Name:        "Foo.Bar",
			Description: "A test endpoint",
			Handler:     "meta",
			Host:        []string{"foo.com"},
			Method:      []string{"GET"},
			Path:        []string{"/test"},
		},
	}

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

	for _, d := range testData {
		// encode
		e := Encode(d)
		// decode
		de := Decode(e)

		// nil endpoint returns nil
		if d == nil {
			if e != nil {
				t.Fatalf("expected nil got %v", e)
			}
			if de != nil {
				t.Fatalf("expected nil got %v", de)
			}

			continue
		}

		// check encoded map
		name := e["endpoint"]
		desc := e["description"]
		method := strings.Split(e["method"], ",")
		path := strings.Split(e["path"], ",")
		host := strings.Split(e["host"], ",")
		handler := e["handler"]

		if name != d.Name {
			t.Fatalf("expected %v got %v", d.Name, name)
		}
		if desc != d.Description {
			t.Fatalf("expected %v got %v", d.Description, desc)
		}
		if handler != d.Handler {
			t.Fatalf("expected %v got %v", d.Handler, handler)
		}
		if ok := compare(d.Method, method); !ok {
			t.Fatalf("expected %v got %v", d.Method, method)
		}
		if ok := compare(d.Path, path); !ok {
			t.Fatalf("expected %v got %v", d.Path, path)
		}
		if ok := compare(d.Host, host); !ok {
			t.Fatalf("expected %v got %v", d.Host, host)
		}

		if de.Name != d.Name {
			t.Fatalf("expected %v got %v", d.Name, de.Name)
		}
		if de.Description != d.Description {
			t.Fatalf("expected %v got %v", d.Description, de.Description)
		}
		if de.Handler != d.Handler {
			t.Fatalf("expected %v got %v", d.Handler, de.Handler)
		}
		if ok := compare(d.Method, de.Method); !ok {
			t.Fatalf("expected %v got %v", d.Method, de.Method)
		}
		if ok := compare(d.Path, de.Path); !ok {
			t.Fatalf("expected %v got %v", d.Path, de.Path)
		}
		if ok := compare(d.Host, de.Host); !ok {
			t.Fatalf("expected %v got %v", d.Host, de.Host)
		}
	}
}

func TestValidate(t *testing.T) {
	epPcre := &Endpoint{
		Name:        "Foo.Bar",
		Description: "A test endpoint",
		Handler:     "meta",
		Host:        []string{"foo.com"},
		Method:      []string{"GET"},
		Path:        []string{"^/test/?$"},
	}
	if err := Validate(epPcre); err != nil {
		t.Fatal(err)
	}

	epGpath := &Endpoint{
		Name:        "Foo.Bar",
		Description: "A test endpoint",
		Handler:     "meta",
		Host:        []string{"foo.com"},
		Method:      []string{"GET"},
		Path:        []string{"/test/{id}"},
	}
	if err := Validate(epGpath); err != nil {
		t.Fatal(err)
	}

	epPcreInvalid := &Endpoint{
		Name:        "Foo.Bar",
		Description: "A test endpoint",
		Handler:     "meta",
		Host:        []string{"foo.com"},
		Method:      []string{"GET"},
		Path:        []string{"/test/?$"},
	}
	if err := Validate(epPcreInvalid); err == nil {
		t.Fatalf("invalid pcre %v", epPcreInvalid.Path[0])
	}

}

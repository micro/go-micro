package client

import (
	"bytes"
	"testing"
)

func TestTemplates(t *testing.T) {
	name := "foo"
	version := "123"
	typ := "service"
	namespace := "default"

	// Render default service
	s := NewService(name, version, typ, namespace)
	bs := new(bytes.Buffer)
	if err := renderTemplate(templates["service"], bs, s); err != nil {
		t.Errorf("Failed to render kubernetes service: %v", err)
	}

	// Render default deployment
	d := NewDeployment(name, version, typ, namespace)
	bd := new(bytes.Buffer)
	if err := renderTemplate(templates["deployment"], bd, d); err != nil {
		t.Errorf("Failed to render kubernetes deployment: %v", err)
	}
}

func TestFormatName(t *testing.T) {
	testCases := []struct {
		name   string
		expect string
	}{
		{"foobar", "foobar"},
		{"foo-bar", "foo-bar"},
		{"foo.bar", "foo-bar"},
		{"Foo.Bar", "foo-bar"},
		{"go.micro.foo.bar", "go-micro-foo-bar"},
	}

	for _, test := range testCases {
		v := Format(test.name)
		if v != test.expect {
			t.Fatalf("Expected name %s for %s got: %s", test.expect, test.name, v)
		}
	}
}

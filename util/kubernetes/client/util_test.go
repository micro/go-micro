package client

import (
	"bytes"
	"testing"

	"github.com/micro/go-micro/v3/runtime"
)

func TestTemplates(t *testing.T) {
	srv := &runtime.Service{Name: "foo", Version: "123"}
	opts := &runtime.CreateOptions{Type: "service", Namespace: "default"}

	// Render default service
	s := NewService(srv, opts)
	bs := new(bytes.Buffer)
	if err := renderTemplate(templates["service"], bs, s); err != nil {
		t.Errorf("Failed to render kubernetes service: %v", err)
	}

	// Render default deployment
	d := NewDeployment(srv, opts)
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
		{"go.micro.foo.bar", "go-micro-foo-bar"},
		{"foo/bar_baz", "foo-bar-baz"},
	}

	for _, test := range testCases {
		v := Format(test.name)
		if v != test.expect {
			t.Fatalf("Expected name %s for %s got: %s", test.expect, test.name, v)
		}
	}
}

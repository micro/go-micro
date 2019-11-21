package client

import (
	"bytes"
	"testing"
)

func TestTemplates(t *testing.T) {
	name := "foo"
	version := "1.2.3"
	source := "github.com/foo/bar"

	// Render default service
	s := DefaultService(name, version)
	bs := new(bytes.Buffer)
	if err := renderTemplate(templates["service"], bs, s); err != nil {
		t.Errorf("Failed to render kubernetes service: %v", err)
	}

	// Render default deployment
	d := DefaultDeployment(name, version, source)
	bd := new(bytes.Buffer)
	if err := renderTemplate(templates["deployment"], bd, d); err != nil {
		t.Errorf("Failed to render kubernetes deployment: %v", err)
	}
}

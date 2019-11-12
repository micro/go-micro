package client

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestTemplates(t *testing.T) {
	name := "foo"
	version := "1.2.3"

	// Render default service
	s := DefaultService(name, version)
	bs := new(bytes.Buffer)
	path := filepath.Join("internal", "templates", "service.yaml.tmpl")
	if err := renderTemplateFile(path, bs, s); err != nil {
		t.Errorf("Failed to render kubernetes service: %v", err)
	}

	// Render default deployment
	d := DefaultDeployment(name, version)
	bd := new(bytes.Buffer)
	path = filepath.Join("internal", "templates", "deployment.yaml.tmpl")
	if err := renderTemplateFile(path, bd, d); err != nil {
		t.Errorf("Failed to render kubernetes deployment: %v", err)
	}
}

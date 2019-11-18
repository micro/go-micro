package client

import (
	"bytes"
	"testing"
)

func TestTemplates(t *testing.T) {
	name := "foo"
	version := "1.2.3"

	// Render default service
	s := DefaultService(name, version)
	bs := new(bytes.Buffer)
	if err := renderTemplate(serviceTmpl, bs, s); err != nil {
		t.Errorf("Failed to render kubernetes service: %v", err)
	}

	// Render default deployment
	d := DefaultDeployment(name, version)
	bd := new(bytes.Buffer)
	if err := renderTemplate(deploymentTmpl, bd, d); err != nil {
		t.Errorf("Failed to render kubernetes deployment: %v", err)
	}
}

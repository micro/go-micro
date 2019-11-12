package template

import (
	"testing"

	"github.com/micro/go-micro/runtime/kubernetes/client"
)

func TestTemplates(t *testing.T) {
	name := "foo"
	version := "1.2.3"

	s := client.DefaultService(name, version)
	_, err := RenderService(s)
	if err != nil {
		t.Errorf("Failed to render kubernetes service: %v", err)
	}

	d := client.DefaultDeployment(name, version)
	_, err = RenderDeployment(d)
	if err != nil {
		t.Errorf("failed to render kubernetes deployment: %v", err)
	}
}

package template

import (
	"bytes"
	"io"
	"text/template"

	"github.com/micro/go-micro/runtime/kubernetes/client"
)

func executeTemplateFile(path string, data interface{}) (io.Reader, error) {
	t, err := template.ParseFiles(path)
	if err != nil {
		return nil, err
	}

	b := new(bytes.Buffer)
	if err := t.Execute(b, data); err != nil {
		return nil, err
	}

	return b, nil
}

// RenderDeployment returns a rendered body of kubernetes deployment YAML manifest
func RenderDeployment(d *client.Deployment) (io.Reader, error) {
	return executeTemplateFile("deployment.yaml.tmpl", d)
}

// RenderService returns a rendered body of kubernetes service YAML manifest
func RenderService(s *client.Service) (io.Reader, error) {
	return executeTemplateFile("service.yaml.tmpl", s)
}

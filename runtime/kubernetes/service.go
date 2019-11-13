package kubernetes

import (
	"io"
	"strings"

	"github.com/micro/go-micro/runtime"
	"github.com/micro/go-micro/runtime/kubernetes/client"
)

type service struct {
	// service to manage
	*runtime.Service
	// output for logs
	output io.Writer
	// Kubernetes service
	kservice *client.Service
	// Kubernetes deployment
	kdeploy *client.Deployment
}

func newService(s *runtime.Service, c runtime.CreateOptions) *service {
	kservice := client.DefaultService(s.Name, s.Version)
	kdeploy := client.DefaultDeployment(s.Name, s.Version)

	env := make([]client.EnvVar, 0, len(c.Env))
	for _, evar := range c.Env {
		evarPair := strings.Split(evar, "=")
		env = append(env, client.EnvVar{Name: evarPair[0], Value: evarPair[1]})
	}

	// if environment has been supplied update deployment
	if len(env) > 0 {
		kdeploy.Spec.Template.PodSpec.Containers[0].Env = env
	}

	// if Command has been supplied override the default command
	if len(c.Command) > 0 {
		kdeploy.Spec.Template.PodSpec.Containers[0].Command = c.Command
	}

	return &service{
		Service:  s,
		kservice: kservice,
		kdeploy:  kdeploy,
		output:   c.Output,
	}
}

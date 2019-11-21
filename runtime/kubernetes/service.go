package kubernetes

import (
	"strings"

	"github.com/micro/go-micro/runtime"
	"github.com/micro/go-micro/runtime/kubernetes/client"
	"github.com/micro/go-micro/util/log"
)

type service struct {
	// service to manage
	*runtime.Service
	// Kubernetes service
	kservice *client.Service
	// Kubernetes deployment
	kdeploy *client.Deployment
}

func newService(s *runtime.Service, c runtime.CreateOptions) *service {
	kservice := client.DefaultService(s.Name, s.Version)
	kdeploy := client.DefaultDeployment(s.Name, s.Version, s.Source)

	env := make([]client.EnvVar, 0, len(c.Env))
	for _, evar := range c.Env {
		evarPair := strings.Split(evar, "=")
		env = append(env, client.EnvVar{Name: evarPair[0], Value: evarPair[1]})
	}

	// TODO: should we append instead of overriding?
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
	}
}

func deploymentResource(d *client.Deployment) *client.Resource {
	return &client.Resource{
		Name:  d.Metadata.Name,
		Kind:  "deployment",
		Value: d,
	}
}

func serviceResource(s *client.Service) *client.Resource {
	return &client.Resource{
		Name:  s.Metadata.Name,
		Kind:  "service",
		Value: s,
	}
}

// Start starts the Kubernetes service. It creates new kubernetes deployment and service API objects
func (s *service) Start(k client.Kubernetes) error {
	// create deployment first; if we fail, we dont create service
	if err := k.Create(deploymentResource(s.kdeploy)); err != nil {
		log.Debugf("Runtime failed to create deployment: %v", err)
		return err
	}
	// create service now that the deployment has been created
	if err := k.Create(serviceResource(s.kservice)); err != nil {
		log.Debugf("Runtime failed to create service: %v", err)
		return err
	}

	return nil
}

func (s *service) Stop(k client.Kubernetes) error {
	// first attempt to delete service
	if err := k.Delete(serviceResource(s.kservice)); err != nil {
		log.Debugf("Runtime failed to delete service: %v", err)
		return err
	}
	// delete deployment once the service has been deleted
	if err := k.Delete(deploymentResource(s.kdeploy)); err != nil {
		log.Debugf("Runtime failed to delete deployment: %v", err)
		return err
	}

	return nil
}

func (s *service) Update(k client.Kubernetes) error {
	if err := k.Update(deploymentResource(s.kdeploy)); err != nil {
		log.Debugf("Runtime failed to update deployment: %v", err)
		return err
	}
	if err := k.Update(serviceResource(s.kservice)); err != nil {
		log.Debugf("Runtime failed to update service: %v", err)
		return err
	}

	return nil
}

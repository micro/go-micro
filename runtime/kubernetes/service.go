package kubernetes

import (
	"strings"
	"time"

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
	// use pre-formatted name/version
	name := client.Format(s.Name)
	version := client.Format(s.Version)

	kservice := client.NewService(name, version, c.Type)
	kdeploy := client.NewDeployment(name, version, c.Type)

	// attach our values to the deployment; name, version, source
	kdeploy.Metadata.Annotations["name"] = s.Name
	kdeploy.Metadata.Annotations["version"] = s.Version
	kdeploy.Metadata.Annotations["source"] = s.Source

	// associate owner:group to be later augmented
	kdeploy.Metadata.Annotations["owner"] = "micro"
	kdeploy.Metadata.Annotations["group"] = "micro"

	// set a build timestamp to the current time
	if kdeploy.Spec.Template.Metadata.Annotations == nil {
		kdeploy.Spec.Template.Metadata.Annotations = make(map[string]string)
	}
	kdeploy.Spec.Template.Metadata.Annotations["build"] = time.Now().Format(time.RFC3339)

	// define the environment values used by the container
	env := make([]client.EnvVar, 0, len(c.Env))
	for _, evar := range c.Env {
		evarPair := strings.Split(evar, "=")
		env = append(env, client.EnvVar{Name: evarPair[0], Value: evarPair[1]})
	}

	// if environment has been supplied update deployment default environment
	if len(env) > 0 {
		kdeploy.Spec.Template.PodSpec.Containers[0].Env = append(kdeploy.Spec.Template.PodSpec.Containers[0].Env, env...)
	}

	// specify the command to exec
	if len(c.Command) > 0 {
		kdeploy.Spec.Template.PodSpec.Containers[0].Command = c.Command
	} else if len(s.Source) > 0 {
		// default command for our k8s service should be source
		kdeploy.Spec.Template.PodSpec.Containers[0].Command = []string{"go", "run", s.Source}
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

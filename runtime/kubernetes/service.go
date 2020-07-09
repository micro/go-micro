package kubernetes

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/runtime"
	"github.com/micro/go-micro/v2/util/kubernetes/api"
	"github.com/micro/go-micro/v2/util/kubernetes/client"
)

type service struct {
	// service to manage
	*runtime.Service
	// Kubernetes service
	kservice *client.Service
	// Kubernetes deployment
	kdeploy *client.Deployment
}

func parseError(err error) *api.Status {
	status := new(api.Status)
	json.Unmarshal([]byte(err.Error()), &status)
	return status
}

func newService(s *runtime.Service, c runtime.CreateOptions) *service {
	// use pre-formatted name/version
	name := client.Format(s.Name)
	version := client.Format(s.Version)

	kservice := client.NewService(name, version, c.Type, c.Namespace)
	kdeploy := client.NewDeployment(name, version, c.Type, c.Namespace)

	// ensure the metadata is set
	if kdeploy.Spec.Template.Metadata.Annotations == nil {
		kdeploy.Spec.Template.Metadata.Annotations = make(map[string]string)
	}

	// create if non existent
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}

	// add the service metadata to the k8s labels, do this first so we
	// don't override any labels used by the runtime, e.g. name
	for k, v := range s.Metadata {
		kdeploy.Metadata.Annotations[k] = v
	}

	// attach our values to the deployment; name, version, source
	kdeploy.Metadata.Annotations["name"] = s.Name
	kdeploy.Metadata.Annotations["version"] = s.Version
	kdeploy.Metadata.Annotations["source"] = s.Source

	// associate owner:group to be later augmented
	kdeploy.Metadata.Annotations["owner"] = "micro"
	kdeploy.Metadata.Annotations["group"] = "micro"

	// update the deployment is a custom source is provided
	if len(c.Image) > 0 {
		for i := range kdeploy.Spec.Template.PodSpec.Containers {
			kdeploy.Spec.Template.PodSpec.Containers[i].Image = c.Image
			kdeploy.Spec.Template.PodSpec.Containers[i].Command = []string{}
			kdeploy.Spec.Template.PodSpec.Containers[i].Args = []string{}
		}
	}

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

	// set the command if specified
	if len(c.Command) > 0 {
		kdeploy.Spec.Template.PodSpec.Containers[0].Command = c.Command
	}

	if len(c.Args) > 0 {
		kdeploy.Spec.Template.PodSpec.Containers[0].Args = c.Args
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
func (s *service) Start(k client.Client, opts ...client.CreateOption) error {
	// create deployment first; if we fail, we dont create service
	if err := k.Create(deploymentResource(s.kdeploy), opts...); err != nil {
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Runtime failed to create deployment: %v", err)
		}
		s.Status("error", err)
		v := parseError(err)
		if v.Reason == "AlreadyExists" {
			return runtime.ErrAlreadyExists
		}
		return err
	}
	// create service now that the deployment has been created
	if err := k.Create(serviceResource(s.kservice), opts...); err != nil {
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Runtime failed to create service: %v", err)
		}
		s.Status("error", err)
		v := parseError(err)
		if v.Reason == "AlreadyExists" {
			return runtime.ErrAlreadyExists
		}
		return err
	}

	s.Status("started", nil)

	return nil
}

func (s *service) Stop(k client.Client, opts ...client.DeleteOption) error {
	// first attempt to delete service
	if err := k.Delete(serviceResource(s.kservice), opts...); err != nil {
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Runtime failed to delete service: %v", err)
		}
		s.Status("error", err)
		return err
	}
	// delete deployment once the service has been deleted
	if err := k.Delete(deploymentResource(s.kdeploy), opts...); err != nil {
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Runtime failed to delete deployment: %v", err)
		}
		s.Status("error", err)
		return err
	}

	s.Status("stopped", nil)

	return nil
}

func (s *service) Update(k client.Client, opts ...client.UpdateOption) error {
	if err := k.Update(deploymentResource(s.kdeploy), opts...); err != nil {
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Runtime failed to update deployment: %v", err)
		}
		s.Status("error", err)
		return err
	}
	if err := k.Update(serviceResource(s.kservice), opts...); err != nil {
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Runtime failed to update service: %v", err)
		}
		return err
	}

	return nil
}

func (s *service) Status(status string, err error) {
	s.Metadata["lastStatusUpdate"] = time.Now().Format(time.RFC3339)
	if err == nil {
		s.Metadata["status"] = status
		delete(s.Metadata, "error")
		return
	}
	s.Metadata["status"] = "error"
	s.Metadata["error"] = err.Error()
}

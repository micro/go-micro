package kubernetes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/runtime"
	"github.com/micro/go-micro/v3/util/kubernetes/api"
	"github.com/micro/go-micro/v3/util/kubernetes/client"
)

// getServices queries kubernetes for services. It gets information from both the pods and the
// deployments
func (k *kubernetes) getServices(opts ...client.GetOption) ([]*runtime.Service, error) {
	// get the deployments
	depList := new(client.DeploymentList)
	d := &client.Resource{
		Kind:  "deployment",
		Value: depList,
	}
	if err := k.client.Get(d, opts...); err != nil {
		return nil, err
	}

	srvMap := make(map[string]*runtime.Service, len(depList.Items))

	// loop through the services and create a deployment for each
	for _, kdep := range depList.Items {
		srv := &runtime.Service{
			Name:     kdep.Metadata.Labels["name"],
			Version:  kdep.Metadata.Labels["version"],
			Source:   kdep.Metadata.Labels["source"],
			Metadata: kdep.Metadata.Annotations,
		}

		// this metadata was injected by the k8s runtime
		delete(srv.Metadata, "name")
		delete(srv.Metadata, "version")
		delete(srv.Metadata, "source")

		// parse out deployment status and inject into service metadata
		if len(kdep.Status.Conditions) > 0 {
			srv.Status = transformStatus(kdep.Status.Conditions[0].Type)
			srv.Metadata["started"] = kdep.Status.Conditions[0].LastUpdateTime
		} else {
			srv.Status = runtime.Unknown
		}

		srvMap[resourceName(srv)] = srv
	}

	// get the pods from k8s
	podList := new(client.PodList)
	p := &client.Resource{
		Kind:  "pod",
		Value: podList,
	}
	if err := k.client.Get(p, opts...); err != nil {
		logger.Errorf("Error fetching pods: %v", err)
		return nil, nil
	}

	for _, item := range podList.Items {
		// skip if we can't get the container
		if len(item.Status.Containers) == 0 {
			continue
		}

		// lookup the service in the map
		key := resourceName(&runtime.Service{
			Name:    item.Metadata.Labels["name"],
			Version: item.Metadata.Labels["version"],
		})
		srv, ok := srvMap[key]
		if !ok {
			continue
		}

		// use the pod status over the deployment status (contains more details)
		srv.Status = transformStatus(item.Status.Phase)

		// set start time
		state := item.Status.Containers[0].State
		if state.Running != nil {
			srv.Metadata["started"] = state.Running.Started
		}

		// set status from waiting
		if v := state.Waiting; v != nil {
			srv.Status = runtime.Pending
		}
	}

	// turn the map into an array
	services := make([]*runtime.Service, 0, len(srvMap))
	for _, srv := range srvMap {
		services = append(services, srv)
	}
	return services, nil
}

func (k *kubernetes) createCredentials(service *runtime.Service, options *runtime.CreateOptions) error {
	if len(options.Secrets) == 0 {
		return nil
	}

	data := make(map[string]string, len(options.Secrets))
	for key, value := range options.Secrets {
		data[key] = base64.StdEncoding.EncodeToString([]byte(value))
	}

	// construct the k8s secret object
	secret := &client.Secret{
		Type: "Opaque",
		Data: data,
		Metadata: &client.Metadata{
			Name:      resourceName(service),
			Namespace: options.Namespace,
		},
	}

	// crete the secret in kubernetes
	err := k.client.Create(&client.Resource{
		Kind:  "secret",
		Name:  resourceName(service),
		Value: secret,
	}, client.CreateNamespace(options.Namespace))

	// ignore the error if the creds already exist
	if err == nil || parseError(err).Reason == "AlreadyExists" {
		return nil
	}

	if logger.V(logger.WarnLevel, logger.DefaultLogger) {
		logger.Warnf("Error generating auth credentials for service: %v", err)
	}
	return err
}

func (k *kubernetes) deleteCredentials(service *runtime.Service, options *runtime.CreateOptions) error {
	// construct the k8s secret object
	secret := &client.Secret{
		Type: "Opaque",
		Metadata: &client.Metadata{
			Name:      resourceName(service),
			Namespace: options.Namespace,
		},
	}

	// crete the secret in kubernetes
	err := k.client.Delete(&client.Resource{
		Kind:  "secret",
		Name:  resourceName(service),
		Value: secret,
	}, client.DeleteNamespace(options.Namespace))

	if err != nil && logger.V(logger.WarnLevel, logger.DefaultLogger) {
		logger.Warnf("Error deleting auth credentials for service: %v", err)
	}

	return err
}

func resourceName(srv *runtime.Service) string {
	return fmt.Sprintf("%v-%v", client.Format(srv.Name), client.Format(srv.Version))
}

// transformStatus takes a deployment status (deploymentcondition.type) and transforms it into a
// runtime service status, e.g. containercreating => starting
func transformStatus(depStatus string) runtime.ServiceStatus {
	switch strings.ToLower(depStatus) {
	case "pending":
		return runtime.Pending
	case "containercreating":
		return runtime.Starting
	case "imagepullbackoff":
		return runtime.Error
	case "crashloopbackoff":
		return runtime.Error
	case "error":
		return runtime.Error
	case "running":
		return runtime.Running
	case "available":
		return runtime.Running
	case "succeeded":
		return runtime.Stopped
	case "failed":
		return runtime.Error
	case "waiting":
		return runtime.Pending
	case "terminated":
		return runtime.Stopped
	default:
		return runtime.Unknown
	}
}

func parseError(err error) *api.Status {
	status := new(api.Status)
	json.Unmarshal([]byte(err.Error()), &status)
	return status
}

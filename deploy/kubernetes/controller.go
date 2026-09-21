package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

// NetworkService is the ClusterIP Service shape owned by the alpha controller.
// Cluster-assigned addresses are deliberately excluded from reconciliation.
type NetworkService struct {
	Name      string
	Namespace string
	Labels    map[string]string
	Selector  map[string]string
	Port      int32
}

func workloadPort(resource Resource) int32 {
	if resource.Spec.Port == 0 {
		return 8080
	}
	return resource.Spec.Port
}

// MapService maps a lifecycle resource to a stable RPC service endpoint.
func MapService(resource Resource) (NetworkService, error) {
	deployment, err := MapDeployment(resource)
	if err != nil {
		return NetworkService{}, err
	}
	return NetworkService{Name: deployment.Name, Namespace: deployment.Namespace, Labels: copyMap(deployment.Labels), Selector: copyMap(deployment.Pod.Labels), Port: workloadPort(resource)}, nil
}

// WorkloadAction contains a Kubernetes API object ready for an adapter to apply.
type WorkloadAction struct {
	Type   ActionType
	Object map[string]interface{}
}

func objectMetadata(resource Resource, name, namespace string, labels map[string]string) map[string]interface{} {
	metadata := map[string]interface{}{"name": name, "namespace": namespace, "labels": labels}
	if resource.UID != "" {
		metadata["ownerReferences"] = []map[string]interface{}{{"apiVersion": Group + "/" + Version, "kind": string(resource.Kind), "name": resource.Name, "uid": resource.UID, "controller": true, "blockOwnerDeletion": true}}
	}
	return metadata
}

// ReconcileWorkloads plans Deployment and Service create/update/noop actions.
// It preserves the original dependency-light Reconcile API for existing callers.
func ReconcileWorkloads(resource Resource, observed Observed) ([]WorkloadAction, []Condition, error) {
	action, conditions, err := Reconcile(resource, observed)
	if err != nil {
		return nil, conditions, err
	}
	deployment := action.Deployment
	service, err := MapService(resource)
	if err != nil {
		return nil, conditions, err
	}
	env := make([]map[string]string, 0, len(deployment.Pod.Container.Environment))
	for _, key := range deployment.Pod.Container.EnvironmentKeys() {
		env = append(env, map[string]string{"name": key, "value": deployment.Pod.Container.Environment[key]})
	}
	container := map[string]interface{}{"name": deployment.Pod.Container.Name, "image": deployment.Pod.Container.Image, "env": env, "ports": []map[string]interface{}{{"name": "rpc", "containerPort": service.Port}}}
	if len(deployment.Pod.Container.Command) > 0 {
		container["command"] = deployment.Pod.Container.Command
	}
	if len(deployment.Pod.Container.Args) > 0 {
		container["args"] = deployment.Pod.Container.Args
	}
	depObject := map[string]interface{}{"apiVersion": "apps/v1", "kind": "Deployment", "metadata": objectMetadata(resource, deployment.Name, deployment.Namespace, deployment.Labels), "spec": map[string]interface{}{"replicas": deployment.Replicas, "selector": map[string]interface{}{"matchLabels": deployment.Pod.Labels}, "template": map[string]interface{}{"metadata": map[string]interface{}{"labels": deployment.Pod.Labels}, "spec": map[string]interface{}{"containers": []map[string]interface{}{container}}}}}
	serviceObject := map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": objectMetadata(resource, service.Name, service.Namespace, service.Labels), "spec": map[string]interface{}{"type": "ClusterIP", "selector": service.Selector, "ports": []map[string]interface{}{{"name": "rpc", "port": service.Port, "targetPort": "rpc", "protocol": "TCP"}}}}
	serviceAction := ActionNoop
	if observed.Service == nil {
		serviceAction = ActionCreate
	} else if !reflect.DeepEqual(*observed.Service, service) {
		serviceAction = ActionUpdate
	}
	if serviceAction != ActionNoop {
		conditions = []Condition{{Type: "Ready", Status: "False", Reason: "Reconciling", Message: "service endpoint is being reconciled"}}
	}
	return []WorkloadAction{{Type: action.Type, Object: depObject}, {Type: serviceAction, Object: serviceObject}}, conditions, nil
}

// WorkloadClient is the cluster adapter for the opt-in controller skeleton.
// Apply should use server-side apply with a stable field manager and preserve
// fields owned by Kubernetes. Observe must return only the fields represented
// by Observed; UpdateStatus writes the CR's status subresource.
type WorkloadClient interface {
	Observe(context.Context, Resource) (Observed, error)
	Apply(context.Context, map[string]interface{}) error
	UpdateStatus(context.Context, Resource, []Condition) error
}

// Controller reconciles one supplied Agent, Service or Flow resource. The host
// supplies watches/requeues and a Kubernetes adapter, keeping client-go out of
// the core framework. No cluster connection or controller starts implicitly.
type Controller struct{ Client WorkloadClient }

func (c Controller) Reconcile(ctx context.Context, resource Resource) ([]Condition, error) {
	if c.Client == nil {
		return nil, errors.New("kubernetes: workload client is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	observed, err := c.Client.Observe(ctx, resource)
	var actions []WorkloadAction
	var conditions []Condition
	if err == nil {
		actions, conditions, err = ReconcileWorkloads(resource, observed)
	}
	if err == nil {
		for _, action := range actions {
			if action.Type != ActionNoop {
				if err = c.Client.Apply(ctx, action.Object); err != nil {
					break
				}
			}
		}
	}
	if err != nil {
		conditions = []Condition{{Type: "Error", Status: "True", Reason: "ReconcileFailed", Message: err.Error()}, {Type: "Ready", Status: "False", Reason: "ReconcileFailed", Message: "workloads have not converged"}}
	}
	if statusErr := c.Client.UpdateStatus(ctx, resource, conditions); statusErr != nil {
		err = errors.Join(err, fmt.Errorf("update status: %w", statusErr))
	}
	return conditions, err
}

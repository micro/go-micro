package kubernetes

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeWorkloadClient struct {
	observed                        Observed
	objects                         []map[string]interface{}
	status                          []Condition
	observeErr, applyErr, statusErr error
}

func (f *fakeWorkloadClient) Observe(context.Context, Resource) (Observed, error) {
	return f.observed, f.observeErr
}
func (f *fakeWorkloadClient) Apply(_ context.Context, object map[string]interface{}) error {
	f.objects = append(f.objects, object)
	return f.applyErr
}
func (f *fakeWorkloadClient) UpdateStatus(_ context.Context, _ Resource, conditions []Condition) error {
	f.status = conditions
	return f.statusErr
}

func TestControllerLifecycle(t *testing.T) {
	for _, kind := range []Kind{KindAgent, KindService, KindFlow} {
		t.Run(string(kind), func(t *testing.T) {
			resource := agentResource()
			resource.Kind, resource.UID, resource.Spec.Port = kind, "resource-uid", 9090
			client := &fakeWorkloadClient{}
			controller := Controller{Client: client}
			conditions, err := controller.Reconcile(context.Background(), resource)
			if err != nil || len(client.objects) != 2 {
				t.Fatalf("create: objects=%v err=%v", client.objects, err)
			}
			if findCondition(conditions, "Ready").Status != "False" {
				t.Fatal("new workloads reported ready")
			}
			deployment, service := client.objects[0], client.objects[1]
			if deployment["apiVersion"] != "apps/v1" || deployment["kind"] != "Deployment" || service["kind"] != "Service" {
				t.Fatal("invalid workload objects")
			}
			metadata := deployment["metadata"].(map[string]interface{})
			owner := metadata["ownerReferences"].([]map[string]interface{})[0]
			if owner["uid"] != resource.UID || owner["kind"] != string(kind) {
				t.Fatalf("owner=%v", owner)
			}
			spec := deployment["spec"].(map[string]interface{})
			template := spec["template"].(map[string]interface{})
			container := template["spec"].(map[string]interface{})["containers"].([]map[string]interface{})[0]
			if container["ports"].([]map[string]interface{})[0]["containerPort"] != int32(9090) {
				t.Fatal("wrong container port")
			}
			serviceSpec := service["spec"].(map[string]interface{})
			if !reflect.DeepEqual(serviceSpec["selector"], template["metadata"].(map[string]interface{})["labels"]) {
				t.Fatal("service does not select workload")
			}
			port := serviceSpec["ports"].([]map[string]interface{})[0]
			if port["port"] != int32(9090) || port["targetPort"] != "rpc" {
				t.Fatalf("service port=%v", port)
			}
			dep, _ := MapDeployment(resource)
			svc, _ := MapService(resource)
			client.observed = Observed{Deployment: &dep, Service: &svc, ReadyReplicas: 2}
			client.objects = nil
			conditions, err = controller.Reconcile(context.Background(), resource)
			if err != nil || len(client.objects) != 0 || findCondition(conditions, "Ready").Status != "True" {
				t.Fatalf("converged: %v %v", conditions, err)
			}
			resource.Spec.Image = "example/support:v2"
			conditions, err = controller.Reconcile(context.Background(), resource)
			if err != nil || len(client.objects) != 1 || findCondition(conditions, "Ready").Status != "False" {
				t.Fatalf("update: %v %v", conditions, err)
			}
		})
	}
}

func TestControllerFailures(t *testing.T) {
	sentinel := errors.New("cluster unavailable")
	for _, stage := range []string{"observe", "apply", "status", "invalid"} {
		t.Run(stage, func(t *testing.T) {
			client := &fakeWorkloadClient{}
			resource := agentResource()
			switch stage {
			case "observe":
				client.observeErr = sentinel
			case "apply":
				client.applyErr = sentinel
			case "status":
				client.statusErr = sentinel
			case "invalid":
				resource.Spec.Port = -1
			}
			_, err := (Controller{Client: client}).Reconcile(context.Background(), resource)
			if err == nil {
				t.Fatal("expected failure")
			}
			if stage != "invalid" && !errors.Is(err, sentinel) {
				t.Fatalf("lost cause: %v", err)
			}
			if stage != "status" && findCondition(client.status, "Error").Status != "True" {
				t.Fatalf("missing error status: %v", client.status)
			}
			if (stage == "observe" || stage == "invalid") && len(client.objects) > 0 {
				t.Fatal("applied after failure")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &fakeWorkloadClient{}
	if _, err := (Controller{Client: client}).Reconcile(ctx, agentResource()); !errors.Is(err, context.Canceled) || len(client.objects) > 0 {
		t.Fatalf("canceled: %v", err)
	}
}

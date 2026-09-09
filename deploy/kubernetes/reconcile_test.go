package kubernetes

import "testing"

func agentResource() Resource {
	return Resource{
		Kind:      KindAgent,
		Name:      "support",
		Namespace: "agents",
		Spec:      WorkloadSpec{Image: "example/support:v1", Replicas: 2, Registry: "kubernetes"},
	}
}

func TestReconcileCreatesWhenAbsent(t *testing.T) {
	action, conds, err := Reconcile(agentResource(), Observed{Deployment: nil})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if action.Type != ActionCreate {
		t.Fatalf("action = %q, want create", action.Type)
	}
	if action.Deployment.Name != "support" || action.Deployment.Replicas != 2 {
		t.Fatalf("desired deployment = %+v", action.Deployment)
	}
	if ready := findCondition(conds, "Ready"); ready == nil || ready.Status != "False" || ready.Reason != "Creating" {
		t.Fatalf("ready condition = %+v, want False/Creating", ready)
	}
}

func TestReconcileNoopWhenMatchedAndReady(t *testing.T) {
	want, _ := MapDeployment(agentResource())
	action, conds, err := Reconcile(agentResource(), Observed{Deployment: &want, ReadyReplicas: 2})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if action.Type != ActionNoop {
		t.Fatalf("action = %q, want noop", action.Type)
	}
	if ready := findCondition(conds, "Ready"); ready == nil || ready.Status != "True" || ready.Reason != "Available" {
		t.Fatalf("ready condition = %+v, want True/Available", ready)
	}
}

func TestReconcileUpdatesOnDrift(t *testing.T) {
	current, _ := MapDeployment(agentResource())
	current.Pod.Container.Image = "example/support:v0" // stale image → drift
	action, _, err := Reconcile(agentResource(), Observed{Deployment: &current, ReadyReplicas: 2})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if action.Type != ActionUpdate {
		t.Fatalf("action = %q, want update", action.Type)
	}
	if action.Deployment.Pod.Container.Image != "example/support:v1" {
		t.Fatalf("update should carry the desired image, got %q", action.Deployment.Pod.Container.Image)
	}
}

func TestReconcileProgressingWhenUnderReplicated(t *testing.T) {
	want, _ := MapDeployment(agentResource())
	_, conds, err := Reconcile(agentResource(), Observed{Deployment: &want, ReadyReplicas: 1})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if ready := findCondition(conds, "Ready"); ready == nil || ready.Status != "False" || ready.Reason != "Progressing" {
		t.Fatalf("ready condition = %+v, want False/Progressing", ready)
	}
}

func TestReconcileErrorOnInvalidSpec(t *testing.T) {
	// Missing image → MapDeployment fails → Error condition, no action.
	_, conds, err := Reconcile(Resource{Kind: KindService, Name: "api"}, Observed{})
	if err == nil {
		t.Fatal("Reconcile should error on an invalid spec")
	}
	if e := findCondition(conds, "Error"); e == nil || e.Status != "True" || e.Reason != "InvalidSpec" {
		t.Fatalf("error condition = %+v, want True/InvalidSpec", e)
	}
}

func findCondition(conds []Condition, typ string) *Condition {
	for i := range conds {
		if conds[i].Type == typ {
			return &conds[i]
		}
	}
	return nil
}

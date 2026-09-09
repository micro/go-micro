package kubernetes

import (
	"fmt"
	"reflect"
)

// Reconcile is the pure decision core an operator's reconcile loop runs: given
// a desired resource and the currently observed cluster state, it computes the
// one action needed to converge (create / update / nothing) plus the status
// conditions to publish. It does not talk to a cluster — no controller-runtime,
// no client-go — so the whole convergence decision is unit-testable. An adapter
// binary supplies Observed from the live cluster and applies the returned
// Action; that adapter is the only piece that needs the Kubernetes client.

// ActionType is the change a reconcile wants applied.
type ActionType string

const (
	// ActionCreate means the workload does not exist yet and should be created.
	ActionCreate ActionType = "create"
	// ActionUpdate means the workload exists but drifts from desired.
	ActionUpdate ActionType = "update"
	// ActionNoop means the workload already matches desired.
	ActionNoop ActionType = "noop"
)

// Action is the change Reconcile decided on, carrying the desired Deployment.
type Action struct {
	Type       ActionType
	Deployment Deployment
}

// Observed is the current cluster state Reconcile compares against. The adapter
// fills it from the live cluster; a nil Deployment means "not created yet".
type Observed struct {
	// Deployment is the workload as it currently exists, or nil if absent.
	Deployment *Deployment
	// ReadyReplicas is how many pods are ready, from the live Deployment status.
	ReadyReplicas int32
}

// Condition is a status condition to publish on the resource — the ready/error
// signal for the inner-loop and deploy story. It mirrors the Kubernetes
// condition shape without importing the API types.
type Condition struct {
	Type    string `json:"type"`   // "Ready" | "Error"
	Status  string `json:"status"` // "True" | "False" | "Unknown"
	Reason  string `json:"reason"`
	Message string `json:"message,omitempty"`
}

// Reconcile computes the action to bring observed toward desired, plus the
// status conditions. A spec that fails to map returns an Error condition and
// the error (no action).
func Reconcile(desired Resource, observed Observed) (Action, []Condition, error) {
	want, err := MapDeployment(desired)
	if err != nil {
		return Action{}, []Condition{{
			Type: "Error", Status: "True", Reason: "InvalidSpec", Message: err.Error(),
		}}, err
	}

	var action Action
	switch {
	case observed.Deployment == nil:
		action = Action{Type: ActionCreate, Deployment: want}
	case deploymentDiffers(*observed.Deployment, want):
		action = Action{Type: ActionUpdate, Deployment: want}
	default:
		action = Action{Type: ActionNoop, Deployment: want}
	}

	return action, conditions(want, observed), nil
}

// conditions derives the Ready condition from observed state against desired.
func conditions(want Deployment, observed Observed) []Condition {
	switch {
	case observed.Deployment == nil:
		return []Condition{{
			Type: "Ready", Status: "False", Reason: "Creating",
			Message: "workload not yet created",
		}}
	case observed.ReadyReplicas < want.Replicas:
		return []Condition{{
			Type: "Ready", Status: "False", Reason: "Progressing",
			Message: fmt.Sprintf("%d/%d replicas ready", observed.ReadyReplicas, want.Replicas),
		}}
	default:
		return []Condition{{
			Type: "Ready", Status: "True", Reason: "Available",
			Message: fmt.Sprintf("%d/%d replicas ready", observed.ReadyReplicas, want.Replicas),
		}}
	}
}

// deploymentDiffers reports whether the observed deployment drifts from desired
// on the fields this operator manages (replicas, container, labels). Fields the
// cluster owns (status, cluster-assigned metadata) are intentionally ignored.
func deploymentDiffers(current, want Deployment) bool {
	return current.Replicas != want.Replicas ||
		!reflect.DeepEqual(current.Pod.Container, want.Pod.Container) ||
		!reflect.DeepEqual(current.Labels, want.Labels)
}

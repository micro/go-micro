package kubernetes

import (
	"strings"
	"testing"
)

func TestCRDManifestsAreStructural(t *testing.T) {
	for _, kind := range []Kind{KindAgent, KindService, KindFlow} {
		manifest := CRDManifests[kind]
		if manifest == "" {
			t.Fatalf("missing manifest for %s", kind)
		}
		checks := []string{
			"apiVersion: apiextensions.k8s.io/v1",
			"kind: CustomResourceDefinition",
			"group: micro.dev",
			"kind: " + string(kind),
			"name: v1alpha1",
			"served: true",
			"storage: true",
			"openAPIV3Schema:",
			"type: object",
			"required: [image]",
		}
		for _, check := range checks {
			if !strings.Contains(manifest, check) {
				t.Fatalf("%s manifest missing %q:\n%s", kind, check, manifest)
			}
		}
	}
}

func TestMapDeploymentForAgent(t *testing.T) {
	deployment, err := MapDeployment(Resource{
		Kind:      KindAgent,
		Name:      "support-agent",
		Namespace: "agents",
		Spec: WorkloadSpec{
			Image:    "ghcr.io/acme/support-agent:v1",
			Replicas: 2,
			Registry: "kubernetes",
			Environment: map[string]string{
				"MODEL": "gpt-5.5",
			},
		},
	})
	if err != nil {
		t.Fatalf("MapDeployment returned error: %v", err)
	}
	if deployment.Name != "support-agent" || deployment.Namespace != "agents" {
		t.Fatalf("unexpected identity: %+v", deployment)
	}
	if deployment.Replicas != 2 {
		t.Fatalf("replicas = %d, want 2", deployment.Replicas)
	}
	if got := deployment.Labels["micro.dev/kind"]; got != "agent" {
		t.Fatalf("micro.dev/kind label = %q, want agent", got)
	}
	container := deployment.Pod.Container
	if container.Image != "ghcr.io/acme/support-agent:v1" {
		t.Fatalf("image = %q", container.Image)
	}
	if got := container.Environment["MICRO_REGISTRY"]; got != "kubernetes" {
		t.Fatalf("MICRO_REGISTRY = %q, want kubernetes", got)
	}
	if got := container.Environment["MODEL"]; got != "gpt-5.5" {
		t.Fatalf("MODEL = %q, want gpt-5.5", got)
	}
}

func TestMapDeploymentDefaultsAndValidation(t *testing.T) {
	deployment, err := MapDeployment(Resource{Kind: KindService, Name: "api", Spec: WorkloadSpec{Image: "api:latest"}})
	if err != nil {
		t.Fatalf("MapDeployment returned error: %v", err)
	}
	if deployment.Namespace != "default" || deployment.Replicas != 1 {
		t.Fatalf("defaults = namespace %q replicas %d", deployment.Namespace, deployment.Replicas)
	}

	if _, err := MapDeployment(Resource{Kind: KindFlow, Name: "ingest"}); err == nil {
		t.Fatal("MapDeployment without image succeeded")
	}
	if _, err := MapDeployment(Resource{Kind: "Job", Name: "job", Spec: WorkloadSpec{Image: "job:latest"}}); err == nil {
		t.Fatal("MapDeployment with unsupported kind succeeded")
	}
}

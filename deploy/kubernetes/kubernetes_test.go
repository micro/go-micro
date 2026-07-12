package kubernetes

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// crdDoc is the minimal structural shape a CRD manifest must have. Parsing the
// embedded YAML into it and asserting the fields is the CI-verifiable
// "manifests validate structurally" check.
type crdDoc struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Group string `yaml:"group"`
		Scope string `yaml:"scope"`
		Names struct {
			Kind   string `yaml:"kind"`
			Plural string `yaml:"plural"`
		} `yaml:"names"`
		Versions []struct {
			Name    string `yaml:"name"`
			Served  bool   `yaml:"served"`
			Storage bool   `yaml:"storage"`
			Schema  struct {
				OpenAPIV3Schema map[string]any `yaml:"openAPIV3Schema"`
			} `yaml:"schema"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

func TestCRDsAreStructurallyValid(t *testing.T) {
	crds, err := CRDs()
	if err != nil {
		t.Fatalf("CRDs: %v", err)
	}
	// One CRD per resource kind.
	wantKinds := map[string]bool{KindAgent: false, KindService: false, KindFlow: false}
	for file, body := range crds {
		var doc crdDoc
		if err := yaml.Unmarshal([]byte(body), &doc); err != nil {
			t.Fatalf("%s: not valid YAML: %v", file, err)
		}
		if doc.APIVersion != "apiextensions.k8s.io/v1" {
			t.Errorf("%s: apiVersion = %q, want apiextensions.k8s.io/v1", file, doc.APIVersion)
		}
		if doc.Kind != "CustomResourceDefinition" {
			t.Errorf("%s: kind = %q, want CustomResourceDefinition", file, doc.Kind)
		}
		if doc.Spec.Group != Group {
			t.Errorf("%s: group = %q, want %q", file, doc.Spec.Group, Group)
		}
		if doc.Spec.Scope != "Namespaced" {
			t.Errorf("%s: scope = %q, want Namespaced", file, doc.Spec.Scope)
		}
		// metadata.name must be <plural>.<group>.
		wantName := doc.Spec.Names.Plural + "." + Group
		if doc.Metadata.Name != wantName {
			t.Errorf("%s: metadata.name = %q, want %q", file, doc.Metadata.Name, wantName)
		}
		if len(doc.Spec.Versions) == 0 {
			t.Fatalf("%s: no versions", file)
		}
		v := doc.Spec.Versions[0]
		if v.Name != Version {
			t.Errorf("%s: version = %q, want %q", file, v.Name, Version)
		}
		if !v.Served || !v.Storage {
			t.Errorf("%s: version must be served and storage (served=%v storage=%v)", file, v.Served, v.Storage)
		}
		if len(v.Schema.OpenAPIV3Schema) == 0 {
			t.Errorf("%s: version has no openAPIV3Schema", file)
		}
		if _, ok := wantKinds[doc.Spec.Names.Kind]; !ok {
			t.Errorf("%s: unexpected names.kind %q", file, doc.Spec.Names.Kind)
			continue
		}
		wantKinds[doc.Spec.Names.Kind] = true
	}
	for kind, seen := range wantKinds {
		if !seen {
			t.Errorf("no CRD manifest for kind %q", kind)
		}
	}
}

func TestRenderAgentToDeployment(t *testing.T) {
	replicas := int32(3)
	r := Resource{
		APIVersion: APIVersion,
		Kind:       KindAgent,
		Metadata:   Metadata{Name: "support", Namespace: "agents", Labels: map[string]string{"team": "cx"}},
		Spec: Spec{
			Image:    "example/support:v1",
			Replicas: &replicas,
			Registry: "nats://registry:4222",
			Port:     8080,
			Env:      map[string]string{"LOG_LEVEL": "debug", "ANTHROPIC_API_KEY": "x"},
			Args:     []string{"run"},
		},
	}
	got, err := Render(r)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	dep := got.Deployment
	if dep.APIVersion != "apps/v1" || dep.Kind != "Deployment" {
		t.Errorf("deployment type = %s/%s, want apps/v1/Deployment", dep.APIVersion, dep.Kind)
	}
	if dep.Metadata.Name != "support" || dep.Metadata.Namespace != "agents" {
		t.Errorf("deployment meta = %+v", dep.Metadata)
	}
	if dep.Spec.Replicas != 3 {
		t.Errorf("replicas = %d, want 3", dep.Spec.Replicas)
	}
	if dep.Metadata.Labels["micro.go-micro.dev/kind"] != KindAgent {
		t.Errorf("kind label = %q, want Agent", dep.Metadata.Labels["micro.go-micro.dev/kind"])
	}
	if dep.Metadata.Labels["app.kubernetes.io/managed-by"] != "go-micro" {
		t.Errorf("managed-by label missing: %v", dep.Metadata.Labels)
	}
	if dep.Metadata.Labels["team"] != "cx" {
		t.Errorf("user label not preserved: %v", dep.Metadata.Labels)
	}
	if got, want := dep.Spec.Selector.MatchLabels["app.kubernetes.io/name"], "support"; got != want {
		t.Errorf("selector = %q, want %q", got, want)
	}
	if len(dep.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("containers = %d, want 1", len(dep.Spec.Template.Spec.Containers))
	}
	c := dep.Spec.Template.Spec.Containers[0]
	if c.Image != "example/support:v1" {
		t.Errorf("image = %q", c.Image)
	}
	if len(c.Ports) != 1 || c.Ports[0].ContainerPort != 8080 {
		t.Errorf("ports = %+v, want containerPort 8080", c.Ports)
	}
	// Registry is surfaced as MICRO_REGISTRY_ADDRESS and env is deterministic.
	if !hasEnv(c.Env, "MICRO_REGISTRY_ADDRESS", "nats://registry:4222") {
		t.Errorf("missing registry env: %+v", c.Env)
	}
	if !hasEnv(c.Env, "MICRO_KIND", "Agent") {
		t.Errorf("missing kind env: %+v", c.Env)
	}
	if !hasEnv(c.Env, "LOG_LEVEL", "debug") {
		t.Errorf("missing user env: %+v", c.Env)
	}

	// Port set → a ClusterIP Service is produced.
	if got.Service == nil {
		t.Fatal("expected a Service when Port is set")
	}
	if got.Service.Spec.Ports[0].Port != 8080 || got.Service.Spec.Selector["app.kubernetes.io/name"] != "support" {
		t.Errorf("service spec = %+v", got.Service.Spec)
	}

	// The manifest round-trips to YAML.
	out, err := got.YAML()
	if err != nil {
		t.Fatalf("YAML: %v", err)
	}
	if !strings.Contains(out, "kind: Deployment") || !strings.Contains(out, "kind: Service") {
		t.Errorf("YAML missing objects:\n%s", out)
	}
}

func TestRenderDefaultsAndNoService(t *testing.T) {
	got, err := Render(Resource{Kind: KindFlow, Metadata: Metadata{Name: "orchestrator"}, Spec: Spec{Image: "example/flow:v1"}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got.Deployment.Spec.Replicas != 1 {
		t.Errorf("default replicas = %d, want 1", got.Deployment.Spec.Replicas)
	}
	if got.Service != nil {
		t.Error("no Service should be produced when Port is unset")
	}
	if got.Deployment.Metadata.Labels["micro.go-micro.dev/kind"] != KindFlow {
		t.Errorf("kind label = %q, want Flow", got.Deployment.Metadata.Labels["micro.go-micro.dev/kind"])
	}
}

func TestRenderValidationErrors(t *testing.T) {
	cases := map[string]Resource{
		"unknown kind":  {Kind: "Widget", Metadata: Metadata{Name: "x"}, Spec: Spec{Image: "i"}},
		"missing kind":  {Metadata: Metadata{Name: "x"}, Spec: Spec{Image: "i"}},
		"missing name":  {Kind: KindService, Spec: Spec{Image: "i"}},
		"missing image": {Kind: KindService, Metadata: Metadata{Name: "x"}},
	}
	for name, r := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Render(r); err == nil {
				t.Fatalf("Render(%+v) = nil error, want validation error", r)
			}
		})
	}
}

func hasEnv(env []EnvVar, name, value string) bool {
	for _, e := range env {
		if e.Name == name {
			return e.Value == value
		}
	}
	return false
}

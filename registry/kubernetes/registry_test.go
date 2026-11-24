package kubernetes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/registry/kubernetes/client"
	"go-micro.dev/v6/registry/kubernetes/client/watch"
)

var _ registry.Registry = (*kregistry)(nil)

// staticClient is an isolated client.Kubernetes for tests that only list and
// patch pods; it records every patch instead of applying it.
type staticClient struct {
	pods    []client.Pod
	patches map[string]*client.Pod
}

func (c *staticClient) ListPods(labels map[string]string) (*client.PodList, error) {
	var items []client.Pod
	for _, p := range c.pods {
		match := true
		for k, v := range labels {
			if got, ok := p.Metadata.Labels[k]; !ok || got == nil || *got != v {
				match = false
				break
			}
		}
		if match {
			items = append(items, p)
		}
	}
	return &client.PodList{Items: items}, nil
}

func (c *staticClient) UpdatePod(name string, pod *client.Pod) (*client.Pod, error) {
	if c.patches == nil {
		c.patches = map[string]*client.Pod{}
	}
	c.patches[name] = pod
	return pod, nil
}

func (c *staticClient) WatchPods(map[string]string) (watch.Watch, error) {
	return nil, errors.New("not supported by staticClient")
}

func strPtr(s string) *string { return &s }

// annotatedPod builds a pod the way Register leaves it: selector labels plus
// the JSON-encoded service in an annotation.
func annotatedPod(name, phase, svcName, svcJSON string) client.Pod {
	return client.Pod{
		Metadata: &client.Meta{
			Name: name,
			Labels: map[string]*string{
				labelTypeKey:                             strPtr(labelTypeValueService),
				svcSelectorPrefix + serviceName(svcName): strPtr(svcSelectorValue),
			},
			Annotations: map[string]*string{
				annotationServiceKeyPrefix + serviceName(svcName): strPtr(svcJSON),
			},
		},
		Status: &client.Status{Phase: phase},
	}
}

// v5Annotation is the exact annotation (json.Marshal of a go-micro.dev/v5
// v5.20.0 registry.Service) written by the micro/plugins v5 kubernetes
// registry. v6 services must read it unchanged so that v5 and v6 services can
// discover each other during a rolling upgrade.
const v5Annotation = `{"name":"greeter","version":"1.2.0","metadata":{"team":"core"},` +
	`"endpoints":[{"request":{"name":"Request","type":"Request","values":[{"name":"name","type":"string","values":null}]},` +
	`"response":{"name":"Response","type":"Response","values":null},"metadata":{"stream":"false"},"name":"Greeter.Hello"}],` +
	`"nodes":[{"metadata":{"protocol":"grpc"},"id":"greeter-abc","address":"10.0.0.7:8080"}]}`

func TestGetServiceDecodesV5Annotation(t *testing.T) {
	r := &kregistry{client: &staticClient{pods: []client.Pod{
		annotatedPod("pod-1", podRunning, "greeter", v5Annotation),
	}}}

	got, err := r.GetService("greeter")
	if err != nil {
		t.Fatalf("GetService: %v", err)
	}

	want := []*registry.Service{{
		Name:     "greeter",
		Version:  "1.2.0",
		Metadata: map[string]string{"team": "core"},
		Endpoints: []*registry.Endpoint{{
			Name: "Greeter.Hello",
			Request: &registry.Value{Name: "Request", Type: "Request", Values: []*registry.Value{
				{Name: "name", Type: "string"},
			}},
			Response: &registry.Value{Name: "Response", Type: "Response"},
			Metadata: map[string]string{"stream": "false"},
		}},
		Nodes: []*registry.Node{{
			Id:       "greeter-abc",
			Address:  "10.0.0.7:8080",
			Metadata: map[string]string{"protocol": "grpc"},
		}},
	}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded service mismatch\n got: %+v\nwant: %+v", got[0], want[0])
	}
}

func TestGetServiceSkipsPodsThatCannotServe(t *testing.T) {
	terminating := annotatedPod("pod-terminating", podRunning, "greeter",
		`{"name":"greeter","version":"1","nodes":[{"id":"terminating","address":"10.0.0.2:80"}]}`)
	terminating.Metadata.DeletionTimestamp = time.Now().UTC().Format(time.RFC3339)

	r := &kregistry{client: &staticClient{pods: []client.Pod{
		annotatedPod("pod-running", podRunning, "greeter",
			`{"name":"greeter","version":"1","nodes":[{"id":"running","address":"10.0.0.1:80"}]}`),
		annotatedPod("pod-pending", "Pending", "greeter",
			`{"name":"greeter","version":"1","nodes":[{"id":"pending","address":"10.0.0.3:80"}]}`),
		terminating,
	}}}

	for name, list := range map[string]func() ([]*registry.Service, error){
		"GetService":   func() ([]*registry.Service, error) { return r.GetService("greeter") },
		"ListServices": func() ([]*registry.Service, error) { return r.ListServices() },
	} {
		services, err := list()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(services) != 1 || len(services[0].Nodes) != 1 || services[0].Nodes[0].Id != "running" {
			t.Fatalf("%s: want only the running pod's node, got %+v", name, services)
		}
	}
}

func TestGetServiceNotFound(t *testing.T) {
	r := &kregistry{client: &staticClient{}}

	if _, err := r.GetService("missing"); !errors.Is(err, registry.ErrNotFound) {
		t.Fatalf("want registry.ErrNotFound, got %v", err)
	}
}

func TestMalformedAnnotation(t *testing.T) {
	r := &kregistry{client: &staticClient{pods: []client.Pod{
		annotatedPod("pod-bad", podRunning, "broken", `{not json`),
		annotatedPod("pod-good", podRunning, "greeter", `{"name":"greeter","version":"1","nodes":[{"id":"n1"}]}`),
	}}}

	if _, err := r.GetService("broken"); err == nil {
		t.Fatal("GetService: want an error for a malformed annotation")
	}

	services, err := r.ListServices()
	if err != nil {
		t.Fatalf("ListServices must skip a malformed annotation, got %v", err)
	}
	if len(services) != 1 || services[0].Name != "greeter" {
		t.Fatalf("ListServices: want only greeter, got %+v", services)
	}
}

func TestRegisterSanitizesServiceNameInKeys(t *testing.T) {
	c := &staticClient{}
	r := &kregistry{client: c}
	t.Setenv("HOSTNAME", "pod-1")

	svc := &registry.Service{
		Name:  "go.micro/srv greeter",
		Nodes: []*registry.Node{{Id: "n1", Address: "10.0.0.1:80"}},
	}
	if err := r.Register(svc); err != nil {
		t.Fatalf("Register: %v", err)
	}

	patch := c.patches["pod-1"]
	if patch == nil {
		t.Fatal("Register did not patch the pod named by HOSTNAME")
	}
	if _, ok := patch.Metadata.Labels[svcSelectorPrefix+"go.micro_srv_greeter"]; !ok {
		t.Fatalf("want a sanitized selector label, got %v", patch.Metadata.Labels)
	}
	if _, ok := patch.Metadata.Annotations[annotationServiceKeyPrefix+"go.micro_srv_greeter"]; !ok {
		t.Fatalf("want a sanitized annotation key, got %v", patch.Metadata.Annotations)
	}

	if err := r.Deregister(svc); err != nil {
		t.Fatalf("Deregister: %v", err)
	}
	for k, v := range c.patches["pod-1"].Metadata.Annotations {
		if v != nil {
			t.Fatalf("Deregister must null annotation %q, got %q", k, *v)
		}
	}
}

func TestRegisterErrors(t *testing.T) {
	r := &kregistry{client: &staticClient{}}
	withNode := &registry.Service{Name: "greeter", Nodes: []*registry.Node{{Id: "n1"}}}

	if err := r.Register(&registry.Service{Name: "greeter"}); !errors.Is(err, ErrNoNodesFound) {
		t.Fatalf("Register without nodes: want ErrNoNodesFound, got %v", err)
	}

	t.Setenv("HOSTNAME", "")
	if err := r.Register(withNode); !errors.Is(err, ErrNoHostname) {
		t.Fatalf("Register without HOSTNAME: want ErrNoHostname, got %v", err)
	}
	if err := r.Deregister(withNode); !errors.Is(err, ErrNoHostname) {
		t.Fatalf("Deregister without HOSTNAME: want ErrNoHostname, got %v", err)
	}
}

func TestNewRegistryWithAddress(t *testing.T) {
	r := NewRegistry(registry.Addrs("http://127.0.0.1:8001"))

	if r.String() != "kubernetes" {
		t.Fatalf("String: want kubernetes, got %q", r.String())
	}
	opts := r.Options()
	if len(opts.Addrs) != 1 || opts.Addrs[0] != "http://127.0.0.1:8001" {
		t.Fatalf("Addrs not kept: %v", opts.Addrs)
	}
	if opts.Timeout != time.Second {
		t.Fatalf("default Timeout: want 1s, got %v", opts.Timeout)
	}

	if err := r.Init(registry.Timeout(3 * time.Second)); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if r.Options().Timeout != 3*time.Second {
		t.Fatalf("Init did not apply Timeout: %v", r.Options().Timeout)
	}
}

func TestWatchReceivesNewMatchingPod(t *testing.T) {
	for _, eventType := range []watch.EventType{watch.Added, watch.Modified} {
		t.Run(string(eventType), func(t *testing.T) {
			pod := annotatedPod("pod-1", podRunning, "greeter", v5Annotation)
			body, err := json.Marshal(pod)
			if err != nil {
				t.Fatal(err)
			}
			event, err := json.Marshal(watch.Event{Type: eventType, Object: body})
			if err != nil {
				t.Fatal(err)
			}
			listed := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Query().Get("watch") == "true" {
					w.WriteHeader(http.StatusOK)
					w.(http.Flusher).Flush()
					select {
					case <-listed:
					case <-r.Context().Done():
						return
					}
					_, _ = w.Write(append(event, '\n'))
					return
				}
				_, _ = w.Write([]byte(`{"items":[]}`))
				close(listed)
			}))
			defer server.Close()

			r := NewRegistry(registry.Addrs(server.URL))
			watcher, err := r.Watch()
			if err != nil {
				t.Fatal(err)
			}
			defer watcher.Stop()
			result, err := watcher.Next()
			if err != nil {
				t.Fatalf("Next: %v", err)
			}
			if result.Action != "create" || result.Service.Name != "greeter" || result.Service.Version != "1.2.0" {
				t.Fatalf("unexpected watch result: %+v", result)
			}
			if len(result.Service.Nodes) != 1 || result.Service.Nodes[0].Id != "greeter-abc" {
				t.Fatalf("lost service nodes: %+v", result.Service.Nodes)
			}
		})
	}
}

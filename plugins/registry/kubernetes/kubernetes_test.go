package kubernetes

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	log "github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/plugins/registry/kubernetes/v3/client"
	"github.com/asim/go-micro/plugins/registry/kubernetes/v3/client/mock"
)

var (
	mockClient = mock.NewClient()
	podIP      = 1
)

func setupPod(name string) *client.Pod {
	p, ok := mockClient.Pods[name]
	if ok || p != nil {
		return p
	}

	// inc pod
	podIP++

	p = &client.Pod{
		Metadata: &client.Meta{
			Name:        name,
			Labels:      make(map[string]*string),
			Annotations: make(map[string]*string),
		},
		Status: &client.Status{
			PodIP: "10.0.0." + strconv.Itoa(podIP),
			Phase: podRunning,
		},
	}

	mockClient.Pods[name] = p
	return p
}

// registers a service against a given pod
func register(r registry.Registry, podName string, svc *registry.Service) {
	os.Setenv("HOSTNAME", podName)

	pod := setupPod(podName)

	svc.Nodes = append(svc.Nodes, &registry.Node{
		Id:       svc.Name + ":" + pod.Metadata.Name,
		Address:  fmt.Sprintf("%s:%d", pod.Status.PodIP, 80),
		Metadata: map[string]string{},
	})

	if err := r.Register(svc); err != nil {
		log.Fatalf("did not expect Register() to fail: %v", err)
	}

	os.Setenv("HOSTNAME", "")
}

func teardownRegistry() {
	mock.Teardown(mockClient)
}

func setupRegistry(opts ...registry.Option) registry.Registry {
	return &kregistry{
		client:  mockClient,
		timeout: time.Second * 1,
	}
}

//
//
// Tests start here
//
//

func TestRegister(t *testing.T) {
	r := setupRegistry()
	defer teardownRegistry()

	svc := &registry.Service{Name: "foo.service", Version: "1"}
	register(r, "pod-1", svc)

	// check pod has correct labels/annotations
	p := mockClient.Pods["pod-1"]
	svcLabel, ok := p.Metadata.Labels[svcSelectorPrefix+"foo.service"]
	if !ok || *svcLabel != svcSelectorValue {
		t.Fatalf("expected to have pod selector label")
	}

	svcData, ok := p.Metadata.Annotations[annotationServiceKeyPrefix+"foo.service"]
	if !ok || len(*svcData) == 0 {
		t.Fatalf("expected to have annotation")
	}

	// unmarshal service data from annotation and compare
	// service passed in to .Register()
	var service *registry.Service
	if err := json.Unmarshal([]byte(*svcData), &service); err != nil {
		t.Fatalf("did not expect register unmarshal to fail %v", err)
	}
	if !reflect.DeepEqual(svc, service) {
		t.Fatal("services did not match")
	}
}

func TestRegisterTwoDifferentServicesOnePod(t *testing.T) {
	r := setupRegistry()
	defer teardownRegistry()

	svc1 := &registry.Service{Name: "foo.service"}
	svc2 := &registry.Service{Name: "bar.service"}

	register(r, "pod-1", svc1)
	register(r, "pod-1", svc2)

	// check pod has correct labels/annotations
	p := mockClient.Pods["pod-1"]
	if svcLabel1, ok := p.Metadata.Labels[svcSelectorPrefix+"foo.service"]; !ok || *svcLabel1 != svcSelectorValue {
		t.Fatalf("expected to have pod selector label for service one")
	}
	if svcLabel2, ok := p.Metadata.Labels[svcSelectorPrefix+"bar.service"]; !ok || *svcLabel2 != svcSelectorValue {
		t.Fatalf("expected to have pod selector label for service two")
	}
	svcData1, ok := p.Metadata.Annotations[annotationServiceKeyPrefix+"foo.service"]
	if !ok || len(*svcData1) == 0 {
		t.Fatalf("expected to have annotation")
	}
	svcData2, ok := p.Metadata.Annotations[annotationServiceKeyPrefix+"bar.service"]
	if !ok || len(*svcData1) == 0 {
		t.Fatalf("expected to have annotation")
	}

	// unmarshal service data from annotation and compare
	// service passed in to .Register()
	var service1 *registry.Service
	if err := json.Unmarshal([]byte(*svcData1), &service1); err != nil {
		t.Fatalf("did not expect register unmarshal to fail %v", err)
	}
	if !reflect.DeepEqual(svc1, service1) {
		t.Fatal("services did not match")
	}

	var service2 *registry.Service
	if err := json.Unmarshal([]byte(*svcData2), &service2); err != nil {
		t.Fatalf("did not expect register unmarshal to fail %v", err)
	}
	if !reflect.DeepEqual(svc2, service2) {
		t.Fatal("services did not match")
	}
}

func TestRegisterTwoDifferentServicesTwoPods(t *testing.T) {
	r := setupRegistry()
	defer teardownRegistry()

	svc1 := &registry.Service{Name: "foo.service"}
	svc2 := &registry.Service{Name: "bar.service"}
	register(r, "pod-1", svc1)
	register(r, "pod-2", svc2)

	// check pod-1 has correct labels/annotations
	p1 := mockClient.Pods["pod-1"]
	if svcLabel1, ok := p1.Metadata.Labels[svcSelectorPrefix+"foo.service"]; !ok || *svcLabel1 != svcSelectorValue {
		t.Fatalf("expected to have pod selector label for foo.service")
	}
	if _, ok := p1.Metadata.Labels[svcSelectorPrefix+"bar.service"]; ok {
		t.Fatal("pod 1 shouldnt have label for bar.service")
	}

	// check pod-2 has correct labels/annotations
	p2 := mockClient.Pods["pod-2"]
	if svcLabel2, ok := p2.Metadata.Labels[svcSelectorPrefix+"bar.service"]; !ok || *svcLabel2 != svcSelectorValue {
		t.Fatalf("expected to have pod selector label for bar.service")
	}
	if _, ok := p2.Metadata.Labels[svcSelectorPrefix+"foo.service"]; ok {
		t.Fatal("pod 2 shouldnt have label for foo.service")
	}

	svcData1, ok := p1.Metadata.Annotations[annotationServiceKeyPrefix+"foo.service"]
	if !ok || len(*svcData1) == 0 {
		t.Fatalf("expected to have annotation")
	}
	if _, okp2 := p1.Metadata.Annotations[annotationServiceKeyPrefix+"bar.service"]; okp2 {
		t.Fatal("bar.service shouldnt have annotation for pod2")
	}

	svcData2, ok := p2.Metadata.Annotations[annotationServiceKeyPrefix+"bar.service"]
	if !ok || len(*svcData2) == 0 {
		t.Fatalf("expected to have annotation")
	}
	if _, okp1 := p2.Metadata.Annotations[annotationServiceKeyPrefix+"foo.service"]; okp1 {
		t.Fatal("bar.service shouldnt have annotation for pod1")
	}

	// unmarshal service data from annotation and compare
	// service passed in to .Register()
	var service1 *registry.Service
	if err := json.Unmarshal([]byte(*svcData1), &service1); err != nil {
		t.Fatalf("did not expect register unmarshal to fail %v", err)
	}
	if !reflect.DeepEqual(svc1, service1) {
		t.Fatal("services did not match")
	}

	var service2 *registry.Service
	if err := json.Unmarshal([]byte(*svcData2), &service2); err != nil {
		t.Fatalf("did not expect register unmarshal to fail %v", err)
	}
	if !reflect.DeepEqual(svc2, service2) {
		t.Fatal("services did not match")
	}
}

func TestRegisterSingleVersionedServiceTwoPods(t *testing.T) {
	r := setupRegistry()
	defer teardownRegistry()

	svc1 := &registry.Service{Name: "foo.service"}
	svc2 := &registry.Service{Name: "foo.service"}
	register(r, "pod-1", svc1)
	register(r, "pod-2", svc2)

	// check pod-1 has correct labels/annotations
	p1 := mockClient.Pods["pod-1"]
	if svcLabel1, ok := p1.Metadata.Labels[svcSelectorPrefix+"foo.service"]; !ok || *svcLabel1 != svcSelectorValue {
		t.Fatalf("expected to have pod selector label for foo.service")
	}

	// check pod-2 has correct labels/annotations
	p2 := mockClient.Pods["pod-2"]
	if svcLabel2, ok := p2.Metadata.Labels[svcSelectorPrefix+"foo.service"]; !ok || *svcLabel2 != svcSelectorValue {
		t.Fatalf("expected to have pod selector label for foo.service")
	}

	svcData1, ok := p1.Metadata.Annotations[annotationServiceKeyPrefix+"foo.service"]
	if !ok || len(*svcData1) == 0 {
		t.Fatalf("expected to have annotation")
	}
	svcData2, ok := p2.Metadata.Annotations[annotationServiceKeyPrefix+"foo.service"]
	if !ok || len(*svcData2) == 0 {
		t.Fatalf("expected to have annotation")
	}

	// unmarshal service data from annotation and compare
	// service passed in to .Register()
	var service1 *registry.Service
	if err := json.Unmarshal([]byte(*svcData1), &service1); err != nil {
		t.Fatalf("did not expect register unmarshal to fail %v", err)
	}
	if !reflect.DeepEqual(svc1, service1) {
		t.Fatal("services did not match")
	}

	var service2 *registry.Service
	if err := json.Unmarshal([]byte(*svcData2), &service2); err != nil {
		t.Fatalf("did not expect register unmarshal to fail %v", err)
	}
	if !reflect.DeepEqual(svc2, service2) {
		t.Fatal("services did not match")
	}
}

func TestDeregister(t *testing.T) {
	r := setupRegistry()
	defer teardownRegistry()

	svc1 := &registry.Service{Name: "foo.service"}
	svc2 := &registry.Service{Name: "foo.service"}
	register(r, "pod-1", svc1)
	register(r, "pod-2", svc2)

	// deregister one service
	os.Setenv("HOSTNAME", "pod-1")
	if err := r.Deregister(svc1); err != nil {
		t.Fatalf("did not expect Deregister to fail %v", err)
	}

	// check pod-1 has correct labels/annotations
	p1 := mockClient.Pods["pod-1"]
	if svcLabel1, ok := p1.Metadata.Labels[svcSelectorPrefix+"foo.service"]; ok && *svcLabel1 == svcSelectorValue {
		t.Fatalf("expected to NOT have pod selector label for foo.service")
	}

	// check pod-2 has correct labels/annotations
	p2 := mockClient.Pods["pod-2"]
	if svcLabel2, ok := p2.Metadata.Labels[svcSelectorPrefix+"foo.service"]; !ok || *svcLabel2 != svcSelectorValue {
		t.Fatalf("expected to have pod selector label for foo.service")
	}

	svcData1, ok := p1.Metadata.Annotations[annotationServiceKeyPrefix+"foo.service"]
	if ok && len(*svcData1) != 0 {
		t.Fatalf("expected to NOT have annotation")
	}
	svcData2, ok := p2.Metadata.Annotations[annotationServiceKeyPrefix+"foo.service"]
	if !ok || len(*svcData2) == 0 {
		t.Fatalf("expected to have annotation")
	}

}

func TestGetService(t *testing.T) {
	r := setupRegistry()
	defer teardownRegistry()

	svc1 := &registry.Service{Name: "foo.service"}
	register(r, "pod-1", svc1)

	service, err := r.GetService("foo.service")
	if err != nil {
		t.Fatalf("did not expect GetService to fail %v", err)
	}

	// compare services
	if !hasServices(service, []*registry.Service{svc1}) {
		t.Fatal("expected service to match")
	}
}

func TestGetServiceSameServiceTwoPods(t *testing.T) {
	r := setupRegistry()
	defer teardownRegistry()

	svc1 := &registry.Service{Name: "foo.service", Version: "1"}
	svc2 := &registry.Service{Name: "foo.service", Version: "1"}
	register(r, "pod-1", svc1)
	register(r, "pod-2", svc2)

	service, err := r.GetService("foo.service")
	if err != nil {
		t.Fatalf("did not expect GetService to fail %v", err)
	}

	if len(service) != 1 {
		t.Fatal("expected there to be only 1 service")
	}

	if len(service[0].Nodes) != 2 {
		t.Fatal("expected there to be 2 nodes")
	}
	if !hasNodes(service[0].Nodes, []*registry.Node{svc1.Nodes[0], svc2.Nodes[0]}) {
		t.Fatal("nodes dont match")
	}
}

func TestGetServiceTwoVersionsTwoPods(t *testing.T) {
	r := setupRegistry()
	defer teardownRegistry()

	svc1 := &registry.Service{Name: "foo.service", Version: "1"}
	svc2 := &registry.Service{Name: "foo.service", Version: "2"}

	register(r, "pod-1", svc1)
	register(r, "pod-2", svc2)

	service, err := r.GetService("foo.service")
	if err != nil {
		t.Fatalf("did not expect GetService to fail %v", err)
	}

	if len(service) != 2 {
		t.Fatal("expected there to be 2 services")
	}

	// compare services
	if !hasServices(service, []*registry.Service{svc1, svc2}) {
		t.Fatal("expected service to match")
	}
}

func TestListServices(t *testing.T) {
	r := setupRegistry()
	defer teardownRegistry()

	svc1 := &registry.Service{Name: "foo.service"}
	svc2 := &registry.Service{Name: "bar.service"}

	register(r, "pod-1", svc1)
	register(r, "pod-2", svc2)

	services, err := r.ListServices()
	if err != nil {
		t.Fatalf("did not expect ListServices to fail %v", err)
	}
	if !hasServices(services, []*registry.Service{
		{Name: "foo.service"},
		{Name: "bar.service"},
	}) {
		t.Fatal("expected services to equal")
	}

	os.Setenv("HOSTNAME", "pod-1")
	r.Deregister(svc1)
	services2, err := r.ListServices()
	if err != nil {
		t.Fatalf("did not expect ListServices to fail %v", err)
	}
	if !hasServices(services2, []*registry.Service{
		{Name: "bar.service"},
	}) {
		t.Fatal("expected services to equal")
	}

	// kill pod without deregistering.
	delete(mockClient.Pods, "pod-2")

	// shoudnt return old data
	services3, err := r.ListServices()
	if err != nil {
		t.Fatalf("did not expect ListServices to fail %v", err)
	}
	if len(services3) != 0 {
		t.Fatal("expected there to be no services")
	}

}

func TestWatcher(t *testing.T) {
	r := setupRegistry()

	// check that service is blank
	if _, err := r.GetService("foo.service"); err != registry.ErrNotFound {
		log.Fatal("expected ErrNotFound")
	}

	// setup svc
	svc1 := &registry.Service{Name: "foo.service", Version: "1"}
	register(r, "pod-1", svc1)

	if routes, err := r.GetService("foo.service"); err != nil {
		t.Fatalf("Querying service returned an error: %v", err)
	} else if len(routes) != 1 {
		t.Fatalf("Expected one route, found %v", len(routes))
	}

	// setup svc
	svc2 := &registry.Service{Name: "foo.service", Version: "1"}
	register(r, "pod-2", svc2)
	time.Sleep(time.Millisecond * 100)

	if routes, err := r.GetService("foo.service"); err != nil {
		t.Fatalf("Querying service returned an error: %v", err)
	} else if len(routes) != 1 {
		t.Fatalf("Expected one service, found %v", len(routes))
	}

	// remove pods
	teardownRegistry()
	time.Sleep(time.Millisecond * 100)

}

func hasNodes(a, b []*registry.Node) bool {
	found := 0
	for _, nodeA := range a {
		for _, nodeB := range b {
			if nodeA.Id == nodeB.Id {
				found++
				break
			}
		}
	}
	return found == len(b)
}

func hasServices(a, b []*registry.Service) bool {
	found := 0

	for _, aV := range a {
		for _, bV := range b {
			if aV.Name != bV.Name {
				continue
			}
			if aV.Version != bV.Version {
				continue
			}
			if !hasNodes(aV.Nodes, bV.Nodes) {
				continue
			}
			found++
			break
		}
	}
	return found == len(b)
}

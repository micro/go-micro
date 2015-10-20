package kubernetes

import (
	"fmt"
	"os"
	"sync"

	"github.com/myodc/go-micro/registry"

	k8s "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

type kregistry struct {
	client    *k8s.Client
	namespace string

	mtx      sync.RWMutex
	services map[string]*registry.Service
}

func (c *kregistry) Deregister(s *registry.Service) error {
	return nil
}

func (c *kregistry) Register(s *registry.Service) error {
	return nil
}

func (c *kregistry) GetService(name string) (*registry.Service, error) {
	c.mtx.RLock()
	svc, ok := c.services[name]
	c.mtx.RUnlock()

	if ok {
		return svc, nil
	}

	selector := labels.SelectorFromSet(labels.Set{"name": name})

	services, err := c.client.Services(c.namespace).List(selector, fields.Everything())
	if err != nil {
		return nil, err
	}

	if len(services.Items) == 0 {
		return nil, fmt.Errorf("Service not found")
	}

	ks := &registry.Service{
		Name: name,
	}

	for _, item := range services.Items {
		ks.Nodes = append(ks.Nodes, &registry.Node{
			Address: item.Spec.ClusterIP,
			Port:    item.Spec.Ports[0].Port,
		})
	}

	return ks, nil
}

func (c *kregistry) ListServices() ([]*registry.Service, error) {
	c.mtx.RLock()
	serviceMap := c.services
	c.mtx.RUnlock()

	var services []*registry.Service

	if len(serviceMap) > 0 {
		for _, service := range serviceMap {
			services = append(services, service)
		}
		return services, nil
	}

	rsp, err := c.client.Services(c.namespace).List(labels.Everything(), fields.Everything())
	if err != nil {
		return nil, err
	}

	for _, svc := range rsp.Items {
		if len(svc.ObjectMeta.Labels["name"]) == 0 {
			continue
		}

		services = append(services, &registry.Service{
			Name: svc.ObjectMeta.Labels["name"],
		})
	}

	return services, nil
}

func (c *kregistry) Watch() (registry.Watcher, error) {
	return newWatcher(c)
}

func NewRegistry(addrs []string, opts ...registry.Option) registry.Registry {
	host := "http://" + os.Getenv("KUBERNETES_RO_SERVICE_HOST") + ":" + os.Getenv("KUBERNETES_RO_SERVICE_PORT")
	if len(addrs) > 0 {
		host = addrs[0]
	}

	client, _ := k8s.New(&k8s.Config{
		Host: host,
	})

	kr := &kregistry{
		client:    client,
		namespace: "default",
		services:  make(map[string]*registry.Service),
	}

	return kr
}

package kubernetes

import (
	"fmt"
	"os"
	"sync"

	"github.com/myodc/go-micro/registry"

	k8s "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
)

type kregistry struct {
	client    *k8s.Client
	namespace string

	mtx      sync.RWMutex
	services map[string]registry.Service
}

func (c *kregistry) Watch() {
	newWatcher(c)
}

func (c *kregistry) Deregister(s registry.Service) error {
	return nil
}

func (c *kregistry) Register(s registry.Service) error {
	return nil
}

func (c *kregistry) GetService(name string) (registry.Service, error) {
	c.mtx.RLock()
	svc, ok := c.services[name]
	c.mtx.RUnlock()

	if ok {
		return svc, nil
	}

	selector := labels.SelectorFromSet(labels.Set{"name": name})

	services, err := c.client.Services(c.namespace).List(selector)
	if err != nil {
		return nil, err
	}

	if len(services.Items) == 0 {
		return nil, fmt.Errorf("Service not found")
	}

	ks := &service{name: name}
	for _, item := range services.Items {
		ks.nodes = append(ks.nodes, &node{
			address: item.Spec.PortalIP,
			port:    item.Spec.Ports[0].Port,
		})
	}

	return ks, nil
}

func (c *kregistry) ListServices() ([]registry.Service, error) {
	c.mtx.RLock()
	serviceMap := c.services
	c.mtx.RUnlock()

	var services []registry.Service

	if len(serviceMap) > 0 {
		for _, service := range serviceMap {
			services = append(services, service)
		}
		return services, nil
	}

	rsp, err := c.client.Services(c.namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, svc := range rsp.Items {
		if len(svc.ObjectMeta.Labels["name"]) == 0 {
			continue
		}

		services = append(services, &service{
			name: svc.ObjectMeta.Labels["name"],
		})
	}

	return services, nil
}

func (c *kregistry) NewService(name string, nodes ...registry.Node) registry.Service {
	var snodes []*node

	for _, nod := range nodes {
		if n, ok := nod.(*node); ok {
			snodes = append(snodes, n)
		}
	}

	return &service{
		name:  name,
		nodes: snodes,
	}
}

func (c *kregistry) NewNode(id, address string, port int) registry.Node {
	return &node{
		id:      id,
		address: address,
		port:    port,
	}
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
		services:  make(map[string]registry.Service),
	}

	kr.Watch()

	return kr
}

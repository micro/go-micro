package kubernetes

import (
	"net"

	"github.com/myodc/go-micro/registry"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

type watcher struct {
	registry *kregistry
	watcher  watch.Interface
}

func (k *watcher) update(event watch.Event) {
	if event.Object == nil {
		return
	}

	var service *api.Service
	switch obj := event.Object.(type) {
	case *api.Service:
		service = obj
	default:
		return
	}

	name, exists := service.ObjectMeta.Labels["name"]
	if !exists {
		return
	}

	switch event.Type {
	case watch.Added, watch.Modified:
	case watch.Deleted:
		k.registry.mtx.Lock()
		delete(k.registry.services, name)
		k.registry.mtx.Unlock()
		return
	default:
		return
	}

	serviceIP := net.ParseIP(service.Spec.ClusterIP)

	k.registry.mtx.Lock()
	k.registry.services[name] = &registry.Service{
		Name: name,
		Nodes: []*registry.Node{
			&registry.Node{
				Address: serviceIP.String(),
				Port:    service.Spec.Ports[0].Port,
			},
		},
	}
	k.registry.mtx.Unlock()
}

func (k *watcher) Stop() {
	k.watcher.Stop()
}

func newWatcher(kr *kregistry) (registry.Watcher, error) {
	svi := kr.client.Services(api.NamespaceAll)

	services, err := svi.List(labels.Everything(), fields.Everything())
	if err != nil {
		return nil, err
	}

	watch, err := svi.Watch(labels.Everything(), fields.Everything(), services.ResourceVersion)
	if err != nil {
		return nil, err
	}

	w := &watcher{
		registry: kr,
		watcher:  watch,
	}

	go func() {
		for event := range watch.ResultChan() {
			w.update(event)
		}
	}()

	return w, nil
}

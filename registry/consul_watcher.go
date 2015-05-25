package registry

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/watch"
)

type consulWatcher struct {
	Registry *consulRegistry
	wp       *watch.WatchPlan
	watchers map[string]*watch.WatchPlan
}

type serviceWatcher struct {
	name string
}

func newConsulWatcher(cr *consulRegistry) *consulWatcher {
	cw := &consulWatcher{
		Registry: cr,
		watchers: make(map[string]*watch.WatchPlan),
	}

	wp, err := watch.Parse(map[string]interface{}{"type": "services"})
	if err == nil {
		wp.Handler = cw.Handle
		go wp.Run(cr.Address)
		cw.wp = wp
	}

	return cw
}

func (cw *consulWatcher) serviceHandler(idx uint64, data interface{}) {
	entries, ok := data.([]*api.ServiceEntry)
	if !ok {
		return
	}

	cs := &Service{}

	for _, e := range entries {
		cs.Name = e.Service.Service
		cs.Nodes = append(cs.Nodes, &Node{
			Id:       e.Service.ID,
			Address:  e.Node.Address,
			Port:     e.Service.Port,
			MetaData: decodeMetaData(e.Service.Tags),
		})
	}

	cw.Registry.mtx.Lock()
	cw.Registry.services[cs.Name] = cs
	cw.Registry.mtx.Unlock()
}

func (cw *consulWatcher) Handle(idx uint64, data interface{}) {
	services, ok := data.(map[string][]string)
	if !ok {
		return
	}

	// add new watchers
	for service, _ := range services {
		if _, ok := cw.watchers[service]; ok {
			continue
		}
		wp, err := watch.Parse(map[string]interface{}{
			"type":    "service",
			"service": service,
		})
		if err == nil {
			wp.Handler = cw.serviceHandler
			go wp.Run(cw.Registry.Address)
			cw.watchers[service] = wp
		}
	}

	cw.Registry.mtx.RLock()
	rservices := cw.Registry.services
	cw.Registry.mtx.RUnlock()

	// remove unknown services from registry
	for service, _ := range rservices {
		if _, ok := services[service]; !ok {
			cw.Registry.mtx.Lock()
			delete(cw.Registry.services, service)
			cw.Registry.mtx.Unlock()
		}
	}

	// remove unknown services from watchers
	for service, w := range cw.watchers {
		if _, ok := services[service]; !ok {
			w.Stop()
			delete(cw.watchers, service)
		}
	}
}

func (cw *consulWatcher) Stop() {
	if cw.wp == nil {
		return
	}
	cw.wp.Stop()
}

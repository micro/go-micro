package registry

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/watch"
)

type ConsulWatcher struct {
	Registry *ConsulRegistry
	wp       *watch.WatchPlan
	watchers map[string]*watch.WatchPlan
}

type serviceWatcher struct {
	name string
}

func (cw *ConsulWatcher) serviceHandler(idx uint64, data interface{}) {
	entries, ok := data.([]*api.ServiceEntry)
	if !ok {
		return
	}

	cs := &ConsulService{}

	for _, e := range entries {
		cs.ServiceName = e.Service.Service
		cs.ServiceNodes = append(cs.ServiceNodes, &ConsulNode{
			Node:        e.Node.Node,
			NodeId:      e.Service.ID,
			NodeAddress: e.Node.Address,
			NodePort:    e.Service.Port,
		})
	}

	cw.Registry.mtx.Lock()
	cw.Registry.services[cs.ServiceName] = cs
	cw.Registry.mtx.Unlock()
}

func (cw *ConsulWatcher) Handle(idx uint64, data interface{}) {
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

func (cw *ConsulWatcher) Stop() {
	if cw.wp == nil {
		return
	}
	cw.wp.Stop()
}

func NewConsulWatcher(cr *ConsulRegistry) *ConsulWatcher {
	cw := &ConsulWatcher{
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

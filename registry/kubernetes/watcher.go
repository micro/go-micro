package kubernetes

import (
	"fmt"
	"net"
	"time"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/proxy/config"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/myodc/go-micro/registry"
)

type watcher struct {
	registry *kregistry
}

func (k *watcher) OnUpdate(services []api.Service) {
	fmt.Println("got update")
	activeServices := util.StringSet{}
	for _, svc := range services {
		fmt.Printf("%#v\n", svc.ObjectMeta)
		name, exists := svc.ObjectMeta.Labels["name"]
		if !exists {
			continue
		}

		activeServices.Insert(name)
		serviceIP := net.ParseIP(svc.Spec.PortalIP)

		ks := &registry.Service{
			Name: name,
			Nodes: []*registry.Node{
				&registry.Node{
					Address: serviceIP.String(),
					Port:    svc.Spec.Ports[0].Port,
				},
			},
		}

		k.registry.mtx.Lock()
		k.registry.services[name] = ks
		k.registry.mtx.Unlock()
	}

	k.registry.mtx.Lock()
	defer k.registry.mtx.Unlock()
	for name, _ := range k.registry.services {
		if !activeServices.Has(name) {
			delete(k.registry.services, name)
		}
	}
}

func newWatcher(kr *kregistry) *watcher {
	serviceConfig := config.NewServiceConfig()
	endpointsConfig := config.NewEndpointsConfig()

	config.NewSourceAPI(
		kr.client.Services(api.NamespaceAll),
		kr.client.Endpoints(api.NamespaceAll),
		time.Second*10,
		serviceConfig.Channel("api"),
		endpointsConfig.Channel("api"),
	)

	ks := &watcher{
		registry: kr,
	}

	serviceConfig.RegisterHandler(ks)
	return ks
}

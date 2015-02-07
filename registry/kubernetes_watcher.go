package registry

import (
	"fmt"
	"net"
	"time"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/proxy/config"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

type KubernetesWatcher struct {
	Registry *KubernetesRegistry
}

func (k *KubernetesWatcher) OnUpdate(services []api.Service) {
	fmt.Println("got update")
	activeServices := util.StringSet{}
	for _, service := range services {
		fmt.Printf("%#v\n", service.ObjectMeta)
		name, exists := service.ObjectMeta.Labels["name"]
		if !exists {
			continue
		}

		activeServices.Insert(name)
		serviceIP := net.ParseIP(service.Spec.PortalIP)

		ks := &KubernetesService{
			ServiceName: name,
			ServiceNodes: []*KubernetesNode{
				&KubernetesNode{
					NodeAddress: serviceIP.String(),
					NodePort:    service.Spec.Port,
				},
			},
		}

		k.Registry.mtx.Lock()
		k.Registry.services[name] = ks
		k.Registry.mtx.Unlock()
	}

	k.Registry.mtx.Lock()
	defer k.Registry.mtx.Unlock()
	for name, _ := range k.Registry.services {
		if !activeServices.Has(name) {
			delete(k.Registry.services, name)
		}
	}
}

func NewKubernetesWatcher(kr *KubernetesRegistry) *KubernetesWatcher {
	serviceConfig := config.NewServiceConfig()
	endpointsConfig := config.NewEndpointsConfig()

	config.NewSourceAPI(
		kr.Client.Services(api.NamespaceAll),
		kr.Client.Endpoints(api.NamespaceAll),
		time.Second*10,
		serviceConfig.Channel("api"),
		endpointsConfig.Channel("api"),
	)

	ks := &KubernetesWatcher{
		Registry: kr,
	}

	serviceConfig.RegisterHandler(ks)
	return ks
}

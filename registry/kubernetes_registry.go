package registry

import (
	"fmt"
	"os"

	k8s "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
)

type KubernetesRegistry struct {
	Client    *k8s.Client
	Namespace string
}

func (c *KubernetesRegistry) Deregister(s Service) error {
	return nil
}

func (c *KubernetesRegistry) Register(s Service) error {
	return nil
}

func (c *KubernetesRegistry) GetService(name string) (Service, error) {
	services, err := c.Client.Services(c.Namespace).List(labels.OneTermEqualSelector("name", name))
	if err != nil {
		return nil, err
	}

	if len(services.Items) == 0 {
		return nil, fmt.Errorf("Service not found")
	}

	ks := &KubernetesService{ServiceName: name}
	for _, item := range services.Items {
		ks.ServiceNodes = append(ks.ServiceNodes, &KubernetesNode{
			NodeAddress: item.Spec.PortalIP,
			NodePort:    item.Spec.Port,
		})
	}

	return ks, nil
}

func (c *KubernetesRegistry) NewService(name string, nodes ...Node) Service {
	var snodes []*KubernetesNode

	for _, node := range nodes {
		if n, ok := node.(*KubernetesNode); ok {
			snodes = append(snodes, n)
		}
	}

	return &KubernetesService{
		ServiceName:  name,
		ServiceNodes: snodes,
	}
}

func (c *KubernetesRegistry) NewNode(id, address string, port int) Node {
	return &KubernetesNode{
		NodeId:      id,
		NodeAddress: address,
		NodePort:    port,
	}
}

func NewKubernetesRegistry() Registry {
	client, _ := k8s.New(&k8s.Config{
		Host: "http://" + os.Getenv("KUBERNETES_RO_SERVICE_HOST") + ":" + os.Getenv("KUBERNETES_RO_SERVICE_PORT"),
	})

	return &KubernetesRegistry{
		Client:    client,
		Namespace: "default",
	}
}

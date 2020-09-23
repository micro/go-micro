package kubernetes

import (
	"github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/runtime"
	"github.com/micro/go-micro/v3/util/kubernetes/client"
)

// createNetworkPolicy creates a networkpolicy resource
func (k *kubernetes) createNetworkPolicy(networkPolicy *runtime.NetworkPolicy) error {
	err := k.client.Create(&client.Resource{
		Kind: "networkpolicy",
		Value: client.NetworkPolicy{
			AllowedLabels: networkPolicy.AllowedLabels,
			Metadata: &client.Metadata{
				Name:      networkPolicy.Name,
				Namespace: networkPolicy.Namespace,
			},
		},
	}, client.CreateNamespace(networkPolicy.Namespace))
	if err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("Error creating resource %s: %v", networkPolicy.ID(), err)
		}
	}
	return err
}

// updateNetworkPolicy updates a networkpolicy resource in-place
func (k *kubernetes) updateNetworkPolicy(networkPolicy *runtime.NetworkPolicy) error {
	err := k.client.Update(&client.Resource{
		Kind: "networkpolicy",
		Value: client.NetworkPolicy{
			AllowedLabels: networkPolicy.AllowedLabels,
			Metadata: &client.Metadata{
				Name:      networkPolicy.Name,
				Namespace: networkPolicy.Namespace,
			},
		},
	}, client.UpdateNamespace(networkPolicy.Namespace))
	if err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("Error updating resource %s: %v", networkPolicy.ID(), err)
		}
	}
	return err
}

// deleteNetworkPolicy deletes a networkpolicy resource
func (k *kubernetes) deleteNetworkPolicy(networkPolicy *runtime.NetworkPolicy) error {
	err := k.client.Delete(&client.Resource{
		Kind: "networkpolicy",
		Value: client.NetworkPolicy{
			AllowedLabels: networkPolicy.AllowedLabels,
			Metadata: &client.Metadata{
				Name:      networkPolicy.Name,
				Namespace: networkPolicy.Namespace,
			},
		},
	}, client.DeleteNamespace(networkPolicy.Namespace))
	if err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("Error deleting resource %s: %v", networkPolicy.ID(), err)
		}
	}
	return err
}

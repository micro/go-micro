package kubernetes

import (
	"strings"

	"github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/util/kubernetes/client"
)

func (k *kubernetes) ensureNamepaceExists(ns string) error {
	namespace := client.Format(ns)
	if namespace == client.DefaultNamespace {
		return nil
	}

	exist, err := k.namespaceExists(namespace)
	if err == nil && exist {
		return nil
	}
	if err != nil {
		if logger.V(logger.WarnLevel, logger.DefaultLogger) {
			logger.Warnf("Error checking namespace %v exists: %v", namespace, err)
		}
		return err
	}

	if err := k.createNamespace(namespace); err != nil {
		if logger.V(logger.WarnLevel, logger.DefaultLogger) {
			logger.Warnf("Error creating namespace %v: %v", namespace, err)
		}
		return err
	}

	return nil
}

// namespaceExists returns a boolean indicating if a namespace exists
func (k *kubernetes) namespaceExists(name string) (bool, error) {
	// populate the cache
	if k.namespaces == nil {
		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Populating namespace cache")
		}

		namespaceList := new(client.NamespaceList)
		resource := &client.Resource{Kind: "namespace", Value: namespaceList}
		if err := k.client.List(resource); err != nil {
			return false, err
		}

		if logger.V(logger.DebugLevel, logger.DefaultLogger) {
			logger.Debugf("Popualted namespace cache successfully with %v items", len(namespaceList.Items))
		}
		k.namespaces = namespaceList.Items
	}

	// check if the namespace exists in the cache
	for _, n := range k.namespaces {
		if n.Metadata.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// createNamespace creates a new k8s namespace
func (k *kubernetes) createNamespace(namespace string) error {
	ns := client.Namespace{Metadata: &client.Metadata{Name: namespace}}
	err := k.client.Create(&client.Resource{Kind: "namespace", Value: ns})

	// ignore err already exists
	if err != nil && strings.Contains(err.Error(), "already exists") {
		logger.Debugf("Ignoring ErrAlreadyExists for namespace %v: %v", namespace, err)
		err = nil
	}

	// add to cache
	if err == nil && k.namespaces != nil {
		k.namespaces = append(k.namespaces, ns)
	}

	return err
}

func (k *kubernetes) CreateNamespace(ns string) error {
	err := k.client.Create(&client.Resource{
		Kind: "namespace",
		Value: client.Namespace{
			Metadata: &client.Metadata{
				Name: ns,
			},
		},
	})
	if err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("Error creating namespace %v: %v", ns, err)
		}
	}
	return err
}

func (k *kubernetes) DeleteNamespace(ns string) error {
	err := k.client.Delete(&client.Resource{
		Kind: "namespace",
		Name: ns,
	})
	if err != nil && logger.V(logger.ErrorLevel, logger.DefaultLogger) {
		logger.Errorf("Error deleting namespace %v: %v", ns, err)
	}
	return err
}

// package kubernetes implements kubernetes micro runtime
package kubernetes

import "github.com/micro/go-micro/runtime"

type kubernetes struct {
}

// Registers a service
func (k *kubernetes) Create(*runtime.Service, ...runtime.CreateOption) error {
	// NOTE: left empty for now
	return nil
}

// Remove a service
func (k *kubernetes) Delete(*runtime.Service) error {
	// NOTE: left empty for now
	return nil
}

// Update the service in place
func (k *kubernetes) Update(*runtime.Service) error {
	// TODO: implement this
	return nil
}

// List the managed services
func (k *kubernetes) List() ([]*runtime.Service, error) {
	// TODO: implement this
	return nil, nil
}

// starts the runtime
func (k *kubernetes) Start() error {
	// NOTE: left empty for now
	return nil
}

// Shutdown the runtime
func (k *kubernetes) Stop() error {
	// NOTE: left empty for now
	return nil
}

package nats_test

import (
	"testing"

	"go-micro.dev/v5/registry"
)

func TestRegister(t *testing.T) {
	service := registry.Service{Name: "test"}
	assertNoError(t, e.registryOne.Register(&service))
	defer e.registryOne.Deregister(&service)

	services, err := e.registryOne.ListServices()
	assertNoError(t, err)
	assertEqual(t, 3, len(services))

	services, err = e.registryTwo.ListServices()
	assertNoError(t, err)
	assertEqual(t, 3, len(services))
}

func TestDeregister(t *testing.T) {
	service1 := registry.Service{Name: "test-deregister", Version: "v1"}
	service2 := registry.Service{Name: "test-deregister", Version: "v2"}

	assertNoError(t, e.registryOne.Register(&service1))
	services, err := e.registryOne.GetService(service1.Name)
	assertNoError(t, err)
	assertEqual(t, 1, len(services))

	assertNoError(t, e.registryOne.Register(&service2))
	services, err = e.registryOne.GetService(service2.Name)
	assertNoError(t, err)
	assertEqual(t, 2, len(services))

	assertNoError(t, e.registryOne.Deregister(&service1))
	services, err = e.registryOne.GetService(service1.Name)
	assertNoError(t, err)
	assertEqual(t, 1, len(services))

	assertNoError(t, e.registryOne.Deregister(&service2))
	services, err = e.registryOne.GetService(service1.Name)
	assertNoError(t, err)
	assertEqual(t, 0, len(services))
}

func TestGetService(t *testing.T) {
	services, err := e.registryTwo.GetService("one")
	assertNoError(t, err)
	assertEqual(t, 1, len(services))
	assertEqual(t, "one", services[0].Name)
	assertEqual(t, 1, len(services[0].Nodes))
}

func TestGetServiceWithNoNodes(t *testing.T) {
	services, err := e.registryOne.GetService("missing")
	assertNoError(t, err)
	assertEqual(t, 0, len(services))
}

func TestGetServiceFromMultipleNodes(t *testing.T) {
	services, err := e.registryOne.GetService("two")
	assertNoError(t, err)
	assertEqual(t, 1, len(services))
	assertEqual(t, "two", services[0].Name)
	assertEqual(t, 2, len(services[0].Nodes))
}

func BenchmarkGetService(b *testing.B) {
	for n := 0; n < b.N; n++ {
		services, err := e.registryTwo.GetService("one")
		assertNoError(b, err)
		assertEqual(b, 1, len(services))
		assertEqual(b, "one", services[0].Name)
	}
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	for n := 0; n < b.N; n++ {
		services, err := e.registryOne.GetService("missing")
		assertNoError(b, err)
		assertEqual(b, 0, len(services))
	}
}

func BenchmarkGetServiceFromMultipleNodes(b *testing.B) {
	for n := 0; n < b.N; n++ {
		services, err := e.registryTwo.GetService("two")
		assertNoError(b, err)
		assertEqual(b, 1, len(services))
		assertEqual(b, "two", services[0].Name)
		assertEqual(b, 2, len(services[0].Nodes))
	}
}

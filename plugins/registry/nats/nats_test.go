package nats_test

import (
	"testing"

	"github.com/micro/go-micro/v2/registry"
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
	t.Skip("not properly implemented")

	service := registry.Service{Name: "test"}

	assertNoError(t, e.registryOne.Register(&service))
	assertNoError(t, e.registryOne.Deregister(&service))

	services, err := e.registryOne.ListServices()
	assertNoError(t, err)
	assertEqual(t, 0, len(services))

	services, err = e.registryTwo.ListServices()
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

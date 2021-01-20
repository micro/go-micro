package eureka

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/hudl/fargo"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/plugins/registry/eureka/v3/mock"
)

func TestRegistration(t *testing.T) {
	testData := []struct {
		getInstanceErr       error
		callCountGetInstance int
		callCountRegister    int
		callCountHeartbeat   int
	}{
		{errors.New("Instance not existing"), 1, 1, 0}, // initial register
		{nil, 1, 0, 1}, // subsequent register
	}

	eureka := NewRegistry().(*eurekaRegistry)

	service := &registry.Service{
		Nodes: []*registry.Node{new(registry.Node)},
	}

	for _, test := range testData {
		mockConn := new(mock.FargoConnection)
		mockConn.GetInstanceReturns(nil, test.getInstanceErr)
		eureka.conn = mockConn

		eureka.Register(service)

		if mockConn.GetInstanceCallCount() != test.callCountGetInstance {
			t.Errorf(
				"Expected exactly %d calls to GetInstance, got %d calls.",
				test.callCountGetInstance,
				mockConn.GetInstanceCallCount(),
			)
		}

		if mockConn.RegisterInstanceCallCount() != test.callCountRegister {
			t.Errorf(
				"Expected exactly %d calls of RegisterInstance, got %d calls.",
				test.callCountRegister,
				mockConn.RegisterInstanceCallCount(),
			)
		}

		if mockConn.HeartBeatInstanceCallCount() != test.callCountHeartbeat {
			t.Errorf(
				"Expected exactly %d calls of HeartBeatInstance, got %d calls.",
				test.callCountHeartbeat,
				mockConn.HeartBeatInstanceCallCount(),
			)
		}
	}
}

func TestSwitchHttpClient(t *testing.T) {
	expected := new(http.Client)

	NewRegistry(func(o *registry.Options) {
		o.Context = context.WithValue(o.Context, contextHttpClient{}, expected)
	})

	if fargo.HttpClient != expected {
		t.Errorf("Unexpected fargo.HttpClient: got %v, want %v", fargo.HttpClient, expected)
	}
}

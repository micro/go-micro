package hystrix

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/asim/go-micro/v3/client"
	merrors "github.com/asim/go-micro/v3/errors"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/selector"
)

func TestBreaker(t *testing.T) {
	// setup
	r := registry.NewMemoryRegistry()
	s := selector.NewSelector(selector.Registry(r))
	c := client.NewClient(
		// set the selector
		client.Selector(s),
		// add the breaker wrapper
		client.Wrap(NewClientWrapper()),
	)

	req := c.NewRequest("test.service", "Test.Method", map[string]string{
		"foo": "bar",
	}, client.WithContentType("application/json"))

	var rsp map[string]interface{}

	// Force to point of trip
	for i := 0; i < (hystrix.DefaultVolumeThreshold * 3); i++ {
		c.Call(context.TODO(), req, rsp)
	}

	err := c.Call(context.TODO(), req, rsp)
	if err == nil {
		t.Error("Expecting tripped breaker, got nil error")
	}

	if err.Error() != "hystrix: circuit open" {
		t.Errorf("Expecting tripped breaker, got %v", err)
	}
}

func TestBreakerWithFilter(t *testing.T) {
	r := registry.NewMemoryRegistry()
	s := selector.NewSelector(selector.Registry(r))
	c := client.NewClient(
		client.Selector(s),
		client.Wrap(NewClientWrapper(WithFilter(func(c context.Context, e error) bool {
			var merr *merrors.Error
			if errors.As(e, &merr) && merr.Detail == "service test.service: not found" {
				return true
			}
			return false
		}))),
	)

	req := c.NewRequest("test.service", "Test.FilterMethod", nil)
	for i := 0; i < (hystrix.DefaultVolumeThreshold); i++ {
		c.Call(context.TODO(), req, nil)
	}

	circuit, _, _ := hystrix.GetCircuit("test.service.Test.FilterMethod")
	if circuit.IsOpen() {
		t.Errorf("breaker should not be opened")
	}

	err := c.Call(context.TODO(), req, nil)
	if err == nil {
		t.Error("original error should be throw out")
	}
}

func TestBreakerWithFallback(t *testing.T) {
	r := registry.NewMemoryRegistry()
	s := selector.NewSelector(selector.Registry(r))
	c := client.NewClient(
		client.Selector(s),
		client.Wrap(NewClientWrapper(WithFallback(func(c context.Context, e error) error {
			var merr *merrors.Error
			if errors.As(e, &merr) && merr.Detail == "service test.service: not found" {
				return hystrix.ErrCircuitOpen
			}
			return e
		}))),
	)

	// trigger fallback to open circuit breaker
	req := c.NewRequest("test.service", "Test.FallbackMethod", nil)
	for i := 0; i < (hystrix.DefaultVolumeThreshold); i++ {
		c.Call(context.TODO(), req, nil)
	}
	err := c.Call(context.TODO(), req, nil)
	if err == nil || !strings.HasPrefix(err.Error(), "fallback failed with 'hystrix: circuit open'") {
		t.Error("fallback-failure error should be throw out")
		return
	}

	circuit, _, _ := hystrix.GetCircuit("test.service.Test.FallbackMethod")
	if !circuit.IsOpen() {
		t.Errorf("breaker should be opened")
	}

	err = c.Call(context.TODO(), req, nil)
	if err == nil {
		t.Error("original error should be throw out")
	}
}

package gobreaker

import (
	"context"
	"testing"

	"github.com/sony/gobreaker"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/errors"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/selector"
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
	for i := 0; i < 6; i++ {
		c.Call(context.TODO(), req, rsp)
	}

	err := c.Call(context.TODO(), req, rsp)
	if err == nil {
		t.Error("Expecting tripped breaker, got nil error")
	}

	merr := err.(*errors.Error)
	if merr.Code != 502 {
		t.Errorf("Expecting tripped breaker, got %v", err)
	}
}

func TestCustomBreaker(t *testing.T) {
	// setup
	r := registry.NewMemoryRegistry()
	s := selector.NewSelector(selector.Registry(r))

	c := client.NewClient(
		// set the selector
		client.Selector(s),
		// add the breaker wrapper
		client.Wrap(NewCustomClientWrapper(
			gobreaker.Settings{},
			BreakService,
		)),
	)

	req := c.NewRequest("test.service", "Test.Method", map[string]string{
		"foo": "bar",
	}, client.WithContentType("application/json"))

	var rsp map[string]interface{}

	// Force to point of trip
	for i := 0; i < 6; i++ {
		c.Call(context.TODO(), req, rsp)
	}

	err := c.Call(context.TODO(), req, rsp)
	if err == nil {
		t.Error("Expecting tripped breaker, got nil error")
	}

	merr := err.(*errors.Error)
	if merr.Code != 502 {
		t.Errorf("Expecting tripped breaker, got %v", err)
	}
}

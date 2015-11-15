package server

import (
	"reflect"
	"testing"

	"github.com/myodc/go-micro/registry"
	"golang.org/x/net/context"
)

type testHandler struct{}

type testRequest struct{}

type testResponse struct{}

func (t *testHandler) Test(ctx context.Context, req *testRequest, rsp *testResponse) error {
	return nil
}

func TestExtractAddress(t *testing.T) {
	data := []struct {
		Input  string
		Output string
	}{
		{"10.0.0.1", "10.0.0.1"},
	}

	for _, d := range data {
		addr, err := extractAddress(d.Input)
		if err != nil {
			t.Errorf("Expected %s: %v", d.Output, err)
		}
		if addr != d.Output {
			t.Errorf("Expected %s, got %s", d.Output, addr)
		}
	}
}

func TestExtractEndpoint(t *testing.T) {
	handler := &testHandler{}
	typ := reflect.TypeOf(handler)

	var endpoints []*registry.Endpoint

	for m := 0; m < typ.NumMethod(); m++ {
		if e := extractEndpoint(typ.Method(m)); e != nil {
			endpoints = append(endpoints, e)
		}
	}

	if i := len(endpoints); i != 1 {
		t.Errorf("Expected 1 endpoint, have %d", i)
	}

	if endpoints[0].Name != "Test" {
		t.Errorf("Expected handler Test, got %s", endpoints[0].Name)
	}

	if endpoints[0].Request == nil {
		t.Error("Expected non nil request")
	}

	if endpoints[0].Response == nil {
		t.Error("Expected non nil request")
	}

	if endpoints[0].Request.Name != "testRequest" {
		t.Errorf("Expected testRequest got %s", endpoints[0].Request.Name)
	}

	if endpoints[0].Response.Name != "testResponse" {
		t.Errorf("Expected testResponse got %s", endpoints[0].Response.Name)
	}

	if endpoints[0].Request.Type != "testRequest" {
		t.Errorf("Expected testRequest type got %s", endpoints[0].Request.Type)
	}

	if endpoints[0].Response.Type != "testResponse" {
		t.Errorf("Expected testResponse type got %s", endpoints[0].Response.Type)
	}

}

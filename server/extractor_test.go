package server

import (
	"context"
	"reflect"
	"testing"

	"github.com/micro/go-micro/v2/registry"
)

type testHandler struct{}

type testRequest struct{}

type testResponse struct{}

func (t *testHandler) Test(ctx context.Context, req *testRequest, rsp *testResponse) error {
	return nil
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

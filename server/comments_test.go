package server

import (
	"context"
	"testing"
)

// TestService is a test service with documented methods
type TestService struct{}

// GetItem retrieves an item by ID. Returns the item if found, error otherwise.
//
// @example {"id": "item-123"}
func (s *TestService) GetItem(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	return nil
}

// CreateItem creates a new item in the system.
//
// @example {"name": "New Item", "value": 42}
func (s *TestService) CreateItem(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	return nil
}

func (s *TestService) NoDoc(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	return nil
}

type TestRequest struct{}
type TestResponse struct{}

func TestExtractHandlerDocs(t *testing.T) {
	handler := &TestService{}
	docs := extractHandlerDocs(handler)

	// Test GetItem extraction
	if docs["GetItem"] == nil {
		t.Fatal("GetItem documentation not extracted")
	}
	if docs["GetItem"]["description"] == "" {
		t.Error("GetItem description is empty")
	}
	if docs["GetItem"]["example"] != `{"id": "item-123"}` {
		t.Errorf("GetItem example = %q, want %q", docs["GetItem"]["example"], `{"id": "item-123"}`)
	}

	// Test CreateItem extraction
	if docs["CreateItem"] == nil {
		t.Fatal("CreateItem documentation not extracted")
	}
	if docs["CreateItem"]["description"] == "" {
		t.Error("CreateItem description is empty")
	}
	if docs["CreateItem"]["example"] != `{"name": "New Item", "value": 42}` {
		t.Errorf("CreateItem example = %q, want %q", docs["CreateItem"]["example"], `{"name": "New Item", "value": 42}`)
	}

	// Test NoDoc (should have no metadata or only empty metadata)
	if docs["NoDoc"] != nil && len(docs["NoDoc"]) > 0 {
		t.Logf("NoDoc metadata: %+v", docs["NoDoc"])
		// Check if all values are empty
		allEmpty := true
		for _, v := range docs["NoDoc"] {
			if v != "" {
				allEmpty = false
				break
			}
		}
		if !allEmpty {
			t.Error("NoDoc should have no metadata with values")
		}
	}
}

func TestNewRpcHandlerAutoExtract(t *testing.T) {
	handler := NewRpcHandler(&TestService{})
	rpcHandler := handler.(*RpcHandler)

	// Check that endpoints have metadata
	var foundGetItem bool
	for _, ep := range rpcHandler.Endpoints() {
		if ep.Name == "TestService.GetItem" {
			foundGetItem = true
			if ep.Metadata["description"] == "" {
				t.Error("GetItem endpoint missing description metadata")
			}
			if ep.Metadata["example"] != `{"id": "item-123"}` {
				t.Errorf("GetItem endpoint example = %q, want %q", ep.Metadata["example"], `{"id": "item-123"}`)
			}
		}
	}

	if !foundGetItem {
		t.Error("GetItem endpoint not found")
	}
}

func TestManualMetadataOverridesAutoExtract(t *testing.T) {
	// Manual metadata should take precedence over auto-extracted
	handler := NewRpcHandler(
		&TestService{},
		WithEndpointDocs(map[string]EndpointDoc{
			"TestService.GetItem": {
				Description: "Manual override description",
				Example:     `{"id": "manual-123"}`,
			},
		}),
	)

	rpcHandler := handler.(*RpcHandler)

	for _, ep := range rpcHandler.Endpoints() {
		if ep.Name == "TestService.GetItem" {
			if ep.Metadata["description"] != "Manual override description" {
				t.Errorf("Manual description not used: got %q", ep.Metadata["description"])
			}
			if ep.Metadata["example"] != `{"id": "manual-123"}` {
				t.Errorf("Manual example not used: got %q", ep.Metadata["example"])
			}
			return
		}
	}

	t.Error("GetItem endpoint not found")
}

func TestWithEndpointScopes(t *testing.T) {
	handler := NewRpcHandler(
		&TestService{},
		WithEndpointScopes("TestService.GetItem", "items:read"),
		WithEndpointScopes("TestService.CreateItem", "items:write", "items:admin"),
	)

	rpcHandler := handler.(*RpcHandler)

	var foundGet, foundCreate bool
	for _, ep := range rpcHandler.Endpoints() {
		switch ep.Name {
		case "TestService.GetItem":
			foundGet = true
			if ep.Metadata["scopes"] != "items:read" {
				t.Errorf("GetItem scopes = %q, want %q", ep.Metadata["scopes"], "items:read")
			}
		case "TestService.CreateItem":
			foundCreate = true
			if ep.Metadata["scopes"] != "items:write,items:admin" {
				t.Errorf("CreateItem scopes = %q, want %q", ep.Metadata["scopes"], "items:write,items:admin")
			}
		}
	}

	if !foundGet {
		t.Error("GetItem endpoint not found")
	}
	if !foundCreate {
		t.Error("CreateItem endpoint not found")
	}
}

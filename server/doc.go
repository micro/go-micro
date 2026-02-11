package server

// Package server provides options for documenting service endpoints.
//
// Documentation is AUTOMATICALLY EXTRACTED from Go doc comments on handler methods.
// You don't need any extra code - just write good comments!
//
// Basic usage (automatic):
//
//	// GetUser retrieves a user by ID from the database.
//	//
//	// @example {"id": "user-123"}
//	func (s *UserService) GetUser(ctx context.Context, req *GetUserRequest, rsp *GetUserResponse) error {
//	    // implementation
//	}
//
//	// Register handler - docs extracted automatically from comments
//	server.Handle(server.NewHandler(new(UserService)))
//
// Advanced usage (manual override):
//
//	// Override auto-extracted docs with manual metadata
//	server.Handle(
//	    server.NewHandler(
//	        new(UserService),
//	        server.WithEndpointDocs(map[string]server.EndpointDoc{
//	            "UserService.GetUser": {
//	                Description: "Custom description overrides comment",
//	                Example: `{"id": "user-123"}`,
//	            },
//	        }),
//	    ),
//	)

// EndpointDoc contains documentation for an endpoint
type EndpointDoc struct {
	Description string // What the endpoint does
	Example     string // Example JSON input
}

// WithEndpointDocs returns a HandlerOption that adds documentation to multiple endpoints.
// This metadata is stored in the registry and used by MCP gateway to generate
// rich tool descriptions for AI agents.
//
// This is a convenience wrapper around EndpointMetadata for adding docs to multiple endpoints at once.
func WithEndpointDocs(docs map[string]EndpointDoc) HandlerOption {
	return func(o *HandlerOptions) {
		if o.Metadata == nil {
			o.Metadata = make(map[string]map[string]string)
		}

		for endpoint, doc := range docs {
			if o.Metadata[endpoint] == nil {
				o.Metadata[endpoint] = make(map[string]string)
			}
			if doc.Description != "" {
				o.Metadata[endpoint]["description"] = doc.Description
			}
			if doc.Example != "" {
				o.Metadata[endpoint]["example"] = doc.Example
			}
		}
	}
}

// WithEndpointDescription is a convenience function for adding a description to a single endpoint.
// For multiple endpoints, use WithEndpointDocs instead.
func WithEndpointDescription(endpoint, description string) HandlerOption {
	return EndpointMetadata(endpoint, map[string]string{
		"description": description,
	})
}

// WithEndpointExample is a convenience function for adding an example to a single endpoint.
func WithEndpointExample(endpoint, example string) HandlerOption {
	return EndpointMetadata(endpoint, map[string]string{
		"example": example,
	})
}

// Package mcp provides Model Context Protocol (MCP) gateway functionality for go-micro services.
// It automatically exposes your microservices as AI-accessible tools through MCP.
//
// Example usage:
//
//	service := micro.NewService(micro.Name("myservice"))
//	service.Init()
//
//	// Add MCP gateway
//	go mcp.Serve(mcp.Options{
//		Registry: service.Options().Registry,
//		Address:  ":3000",
//	})
//
//	service.Run()
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"go-micro.dev/v5/client"
	"go-micro.dev/v5/codec/bytes"
	"go-micro.dev/v5/registry"
)

// Options configures the MCP gateway
type Options struct {
	// Registry for service discovery (required)
	Registry registry.Registry

	// Address to listen on for SSE transport (e.g., ":3000")
	// Leave empty for stdio transport
	Address string

	// Client for making RPC calls (defaults to client.DefaultClient)
	Client client.Client

	// Context for cancellation (defaults to background context)
	Context context.Context

	// Logger for debug output (defaults to log.Default())
	Logger *log.Logger

	// AuthFunc validates requests (optional)
	// Return error to reject, nil to allow
	AuthFunc func(r *http.Request) error
}

// Server represents a running MCP gateway
type Server struct {
	opts     Options
	tools    map[string]*Tool
	toolsMu  sync.RWMutex
	server   *http.Server
	watching bool
}

// Tool represents an MCP tool (exposed service endpoint)
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Service     string                 `json:"-"`
	Endpoint    string                 `json:"-"`
}

// Serve starts an MCP gateway with the given options.
// For stdio transport, leave Address empty.
// For SSE transport, set Address (e.g., ":3000").
func Serve(opts Options) error {
	// Set defaults
	if opts.Client == nil {
		opts.Client = client.DefaultClient
	}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.Logger == nil {
		opts.Logger = log.Default()
	}
	if opts.Registry == nil {
		return fmt.Errorf("registry is required")
	}

	server := &Server{
		opts:  opts,
		tools: make(map[string]*Tool),
	}

	// Discover services and build tool list
	if err := server.discoverServices(); err != nil {
		return fmt.Errorf("failed to discover services: %w", err)
	}

	// Watch for service changes
	go server.watchServices()

	// Start server based on transport
	if opts.Address != "" {
		return server.serveHTTP()
	}
	return server.serveStdio()
}

// ListenAndServe is a convenience function that starts an MCP gateway on the given address.
func ListenAndServe(address string, opts Options) error {
	opts.Address = address
	return Serve(opts)
}

// discoverServices queries the registry and builds the tool list
func (s *Server) discoverServices() error {
	services, err := s.opts.Registry.ListServices()
	if err != nil {
		return err
	}

	s.toolsMu.Lock()
	defer s.toolsMu.Unlock()

	for _, svc := range services {
		// Get full service details
		fullSvcs, err := s.opts.Registry.GetService(svc.Name)
		if err != nil || len(fullSvcs) == 0 {
			continue
		}

		// Convert endpoints to tools
		for _, ep := range fullSvcs[0].Endpoints {
			toolName := fmt.Sprintf("%s.%s", svc.Name, ep.Name)

			// Build input schema from endpoint request type
			inputSchema := s.buildInputSchema(ep.Request)

			s.tools[toolName] = &Tool{
				Name:        toolName,
				Description: fmt.Sprintf("Call %s on %s service", ep.Name, svc.Name),
				InputSchema: inputSchema,
				Service:     svc.Name,
				Endpoint:    ep.Name,
			}
		}
	}

	s.opts.Logger.Printf("[mcp] Discovered %d tools from %d services", len(s.tools), len(services))
	return nil
}

// buildInputSchema converts registry value type information to JSON schema
func (s *Server) buildInputSchema(value *registry.Value) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
	}

	if value == nil || len(value.Values) == 0 {
		return schema
	}

	properties := schema["properties"].(map[string]interface{})
	for _, field := range value.Values {
		properties[field.Name] = map[string]interface{}{
			"type":        s.mapGoTypeToJSON(field.Type),
			"description": fmt.Sprintf("%s field", field.Name),
		}
	}

	return schema
}

// mapGoTypeToJSON maps Go types to JSON schema types
func (s *Server) mapGoTypeToJSON(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "object"
	}
}

// watchServices watches for service registry changes
func (s *Server) watchServices() {
	if s.watching {
		return
	}
	s.watching = true

	watcher, err := s.opts.Registry.Watch()
	if err != nil {
		s.opts.Logger.Printf("[mcp] Failed to watch registry: %v", err)
		return
	}
	defer watcher.Stop()

	for {
		select {
		case <-s.opts.Context.Done():
			return
		default:
			_, err := watcher.Next()
			if err != nil {
				time.Sleep(time.Second)
				continue
			}

			// Rediscover services on any change
			if err := s.discoverServices(); err != nil {
				s.opts.Logger.Printf("[mcp] Failed to rediscover services: %v", err)
			}
		}
	}
}

// serveHTTP starts an HTTP server with SSE transport
func (s *Server) serveHTTP() error {
	mux := http.NewServeMux()

	// MCP endpoints
	mux.HandleFunc("/mcp/tools", s.handleListTools)
	mux.HandleFunc("/mcp/call", s.handleCallTool)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    s.opts.Address,
		Handler: mux,
	}

	s.opts.Logger.Printf("[mcp] MCP gateway listening on %s", s.opts.Address)
	return s.server.ListenAndServe()
}

// serveStdio starts stdio-based MCP server (for Claude Code, etc.)
func (s *Server) serveStdio() error {
	transport := NewStdioTransport(s)
	return transport.Serve()
}

// handleListTools returns the list of available tools
func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	if s.opts.AuthFunc != nil {
		if err := s.opts.AuthFunc(r); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	s.toolsMu.RLock()
	tools := make([]*Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}
	s.toolsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
	})
}

// handleCallTool executes a tool (makes an RPC call)
func (s *Server) handleCallTool(w http.ResponseWriter, r *http.Request) {
	if s.opts.AuthFunc != nil {
		if err := s.opts.AuthFunc(r); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Parse request
	var req struct {
		Tool  string                 `json:"tool"`
		Input map[string]interface{} `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get tool info
	s.toolsMu.RLock()
	tool, exists := s.tools[req.Tool]
	s.toolsMu.RUnlock()

	if !exists {
		http.Error(w, "Tool not found", http.StatusNotFound)
		return
	}

	// Convert input to JSON bytes for RPC call
	inputBytes, err := json.Marshal(req.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Make RPC call
	rpcReq := s.opts.Client.NewRequest(tool.Service, tool.Endpoint, &bytes.Frame{Data: inputBytes})
	var rsp bytes.Frame

	if err := s.opts.Client.Call(r.Context(), rpcReq, &rsp); err != nil {
		s.opts.Logger.Printf("[mcp] RPC call failed: %v", err)
		http.Error(w, fmt.Sprintf("RPC call failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"result": json.RawMessage(rsp.Data),
	})
}

// handleHealth returns gateway health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.toolsMu.RLock()
	toolCount := len(s.tools)
	s.toolsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"tools":  toolCount,
	})
}

// Stop gracefully shuts down the MCP gateway
func (s *Server) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// GetTools returns the current list of available tools
func (s *Server) GetTools() []*Tool {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()

	tools := make([]*Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Example shows how to use the MCP gateway in your code
func Example() {
	// This function is never called - it's just documentation
	_ = func() {
		// In your service code:
		// service := micro.NewService(micro.Name("myservice"))
		// service.Init()

		// Start MCP gateway
		go func() {
			if err := Serve(Options{
				Registry: registry.DefaultRegistry,
				Address:  ":3000",
			}); err != nil {
				log.Fatal(err)
			}
		}()

		// service.Run()
	}
}

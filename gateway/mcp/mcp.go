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
	"strings"
	"sync"
	"time"

	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/codec/bytes"
	"go-micro.dev/v5/metadata"
	"go-micro.dev/v5/registry"

	"github.com/google/uuid"
)

// Metadata keys for MCP tracing and auth propagated via context/metadata.
const (
	// TraceIDKey is the metadata key for the MCP trace ID.
	TraceIDKey = "Mcp-Trace-Id"
	// ToolNameKey is the metadata key for the tool being invoked.
	ToolNameKey = "Mcp-Tool-Name"
	// AccountIDKey is the metadata key for the authenticated account ID.
	AccountIDKey = "Mcp-Account-Id"
)

// AuditRecord represents an immutable log entry for an MCP tool call.
type AuditRecord struct {
	// TraceID uniquely identifies this tool call chain.
	TraceID string `json:"trace_id"`
	// Timestamp of the tool call.
	Timestamp time.Time `json:"timestamp"`
	// Tool is the name of the tool that was called.
	Tool string `json:"tool"`
	// AccountID is the ID of the authenticated account (empty if unauthenticated).
	AccountID string `json:"account_id,omitempty"`
	// Scopes that were required for this tool.
	ScopesRequired []string `json:"scopes_required,omitempty"`
	// Allowed indicates whether the call was authorized.
	Allowed bool `json:"allowed"`
	// Denied reason, if the call was not allowed.
	DeniedReason string `json:"denied_reason,omitempty"`
	// Duration of the RPC call (zero if call was denied before execution).
	Duration time.Duration `json:"duration,omitempty"`
	// Error from the RPC call, if any.
	Error string `json:"error,omitempty"`
}

// AuditFunc is called for every tool call with an audit record.
// Implementations should treat the record as immutable and persist it
// (e.g. to a log, database, or event stream).
type AuditFunc func(record AuditRecord)

// RateLimitConfig configures rate limiting for the MCP gateway.
type RateLimitConfig struct {
	// Requests per second allowed per tool (0 = unlimited).
	RequestsPerSecond float64
	// Burst size (maximum number of requests that can be made at once).
	Burst int
}

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

	// AuthFunc validates requests (optional, legacy)
	// Return error to reject, nil to allow
	AuthFunc func(r *http.Request) error

	// Auth provider for token inspection (optional).
	// When set, incoming requests must carry a Bearer token which is
	// inspected to obtain an account. The account's scopes are then
	// checked against the tool's required scopes.
	Auth auth.Auth

	// AuditFunc is called for every tool call with an immutable audit record.
	// Use this to persist tool-call logs for compliance and debugging.
	AuditFunc AuditFunc

	// RateLimit configures per-tool rate limiting.
	// When set, each tool is limited to the configured requests per second.
	RateLimit *RateLimitConfig
}

// Server represents a running MCP gateway
type Server struct {
	opts     Options
	tools    map[string]*Tool
	toolsMu  sync.RWMutex
	server   *http.Server
	watching bool

	// limiters holds per-tool rate limiters (nil if rate limiting is disabled).
	limiters   map[string]*rateLimiter
	limitersMu sync.RWMutex
}

// Tool represents an MCP tool (exposed service endpoint)
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	// Scopes lists the auth scopes required to call this tool.
	// An empty list means no scope restriction (subject to Auth provider).
	Scopes   []string `json:"scopes,omitempty"`
	Service  string   `json:"-"`
	Endpoint string   `json:"-"`
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
		opts:     opts,
		tools:    make(map[string]*Tool),
		limiters: make(map[string]*rateLimiter),
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

			// Get description from endpoint metadata (set by service during registration)
			description := fmt.Sprintf("Call %s on %s service", ep.Name, svc.Name)
			if ep.Metadata != nil {
				if desc, ok := ep.Metadata["description"]; ok && desc != "" {
					description = desc
				}
			}

			tool := &Tool{
				Name:        toolName,
				Description: description,
				InputSchema: inputSchema,
				Service:     svc.Name,
				Endpoint:    ep.Name,
			}

			// Extract scopes from endpoint metadata
			if ep.Metadata != nil {
				if scopes, ok := ep.Metadata["scopes"]; ok && scopes != "" {
					tool.Scopes = strings.Split(scopes, ",")
				}
			}

			// Add example from metadata if available
			if ep.Metadata != nil {
				if example, ok := ep.Metadata["example"]; ok && example != "" {
					inputSchema["examples"] = []string{example}
				}
			}

			s.tools[toolName] = tool

			// Create rate limiter for this tool if rate limiting is configured
			if s.opts.RateLimit != nil && s.opts.RateLimit.RequestsPerSecond > 0 {
				s.limitersMu.Lock()
				if _, exists := s.limiters[toolName]; !exists {
					s.limiters[toolName] = newRateLimiter(
						s.opts.RateLimit.RequestsPerSecond,
						s.opts.RateLimit.Burst,
					)
				}
				s.limitersMu.Unlock()
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

	// Generate trace ID for this call
	traceID := uuid.New().String()

	// Authenticate and authorise
	var account *auth.Account
	if s.opts.Auth != nil {
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}
		if token == "" {
			s.audit(AuditRecord{TraceID: traceID, Timestamp: time.Now(), Tool: req.Tool, Allowed: false, DeniedReason: "missing token"})
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		acc, err := s.opts.Auth.Inspect(token)
		if err != nil {
			s.audit(AuditRecord{TraceID: traceID, Timestamp: time.Now(), Tool: req.Tool, Allowed: false, DeniedReason: "invalid token"})
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		account = acc

		// Check per-tool scopes
		if len(tool.Scopes) > 0 {
			if !hasScope(account.Scopes, tool.Scopes) {
				s.audit(AuditRecord{
					TraceID: traceID, Timestamp: time.Now(), Tool: req.Tool,
					AccountID: account.ID, ScopesRequired: tool.Scopes,
					Allowed: false, DeniedReason: "insufficient scopes",
				})
				http.Error(w, "Forbidden: insufficient scopes", http.StatusForbidden)
				return
			}
		}
	}

	// Rate limit check
	if err := s.allowRate(req.Tool); err != nil {
		accountID := ""
		if account != nil {
			accountID = account.ID
		}
		s.audit(AuditRecord{
			TraceID: traceID, Timestamp: time.Now(), Tool: req.Tool,
			AccountID: accountID, Allowed: false, DeniedReason: "rate limited",
		})
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Build context with tracing metadata
	ctx := r.Context()
	md := metadata.Metadata{}
	md.Set(TraceIDKey, traceID)
	md.Set(ToolNameKey, req.Tool)
	if account != nil {
		md.Set(AccountIDKey, account.ID)
	}
	ctx = metadata.MergeContext(ctx, md, true)

	// Convert input to JSON bytes for RPC call
	inputBytes, err := json.Marshal(req.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Make RPC call
	start := time.Now()
	rpcReq := s.opts.Client.NewRequest(tool.Service, tool.Endpoint, &bytes.Frame{Data: inputBytes})
	var rsp bytes.Frame

	if err := s.opts.Client.Call(ctx, rpcReq, &rsp); err != nil {
		s.opts.Logger.Printf("[mcp] RPC call failed: %v", err)
		accountID := ""
		if account != nil {
			accountID = account.ID
		}
		s.audit(AuditRecord{
			TraceID: traceID, Timestamp: time.Now(), Tool: req.Tool,
			AccountID: accountID, ScopesRequired: tool.Scopes,
			Allowed: true, Duration: time.Since(start), Error: err.Error(),
		})
		http.Error(w, fmt.Sprintf("RPC call failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Audit successful call
	accountID := ""
	if account != nil {
		accountID = account.ID
	}
	s.audit(AuditRecord{
		TraceID: traceID, Timestamp: time.Now(), Tool: req.Tool,
		AccountID: accountID, ScopesRequired: tool.Scopes,
		Allowed: true, Duration: time.Since(start),
	})

	// Return response with trace ID
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(TraceIDKey, traceID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"result":   json.RawMessage(rsp.Data),
		"trace_id": traceID,
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

// audit emits an audit record if an AuditFunc is configured.
func (s *Server) audit(record AuditRecord) {
	if s.opts.AuditFunc != nil {
		s.opts.AuditFunc(record)
	}
}

// allowRate checks if the tool call is allowed under the configured rate limit.
// Returns nil if allowed, non-nil error if rate-limited.
func (s *Server) allowRate(toolName string) error {
	if s.opts.RateLimit == nil {
		return nil
	}
	s.limitersMu.RLock()
	limiter, ok := s.limiters[toolName]
	s.limitersMu.RUnlock()
	if !ok {
		return nil
	}
	if !limiter.Allow() {
		return fmt.Errorf("rate limit exceeded for tool %s", toolName)
	}
	return nil
}

// hasScope checks if the account has at least one of the required scopes.
func hasScope(accountScopes, requiredScopes []string) bool {
	for _, req := range requiredScopes {
		for _, have := range accountScopes {
			if strings.EqualFold(have, req) {
				return true
			}
		}
	}
	return false
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

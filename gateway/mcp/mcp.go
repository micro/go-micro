// Package mcp provides Model Context Protocol (MCP) gateway functionality for go-micro services.
// It automatically exposes your microservices as AI-accessible tools through MCP.
//
// Example usage:
//
//	service := micro.NewService("myservice", )
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
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v6/auth"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/gateway/schema"
	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
	"go-micro.dev/v6/wrapper/x402"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
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

// mcpInstructions is the system prompt delivered to clients in the initialize
// result. It teaches agents how to discover services, endpoints, and their
// schemas through the micro_ framework tools.
const mcpInstructions = `You are an AI agent operating a Go Micro service mesh.

Tools are named Service.Endpoint (e.g. helloworld.Helloworld.Call). Read each
tool's description and inputSchema (its fields) before calling it.

To discover what is available:
1. Call micro_registry_list to list all registered service names.
2. Call micro_registry_get with a service name to inspect its endpoints,
   including each endpoint's description, example, and request fields.
3. Then call the matching Service.Endpoint tool.

The micro_ prefixed tools (micro_registry_*, micro_store_*, micro_broker_publish)
are framework utilities for service discovery and infrastructure, not service
endpoints.`

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

	// CircuitBreaker configures per-tool circuit breaking.
	// When set, tools that fail repeatedly are temporarily blocked to
	// protect downstream services from cascading failures.
	CircuitBreaker *CircuitBreakerConfig

	// Scopes lets the gateway operator define or override per-tool
	// scope requirements without changing the services themselves.
	// Keys are tool names (e.g. "blog.Blog.Create") and values are the
	// required scopes. When a tool appears in Scopes its scopes
	// replace any scopes declared by the service via endpoint metadata.
	//
	// Example:
	//
	//   Scopes: map[string][]string{
	//       "blog.Blog.Create": {"blog:write"},
	//       "blog.Blog.Delete": {"blog:admin"},
	//   }
	Scopes map[string][]string

	// TraceProvider enables OpenTelemetry tracing for MCP tool calls.
	// When set, each tool call creates a span with attributes for the
	// tool name, account ID, auth outcome, and transport type.
	// Trace context is propagated to downstream RPC calls via metadata.
	//
	// Example:
	//
	//   tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	//   mcp.Serve(mcp.Options{
	//       Registry:      reg,
	//       TraceProvider: tp,
	//   })
	TraceProvider trace.TracerProvider

	// Payment, when set, requires an x402 payment for tool calls
	// (the /mcp/call endpoint). Listing tools and health stay free.
	// Opt-in: leave nil to disable payments.
	Payment *x402.Config

	// ReflectedGRPCTargets exposes unary methods from external gRPC servers
	// that support server reflection as MCP tools. This bridges existing gRPC
	// services into the agent tool catalog without requiring go-micro handlers.
	ReflectedGRPCTargets []ReflectedGRPCTarget
}

// Server represents a running MCP gateway
type Server struct {
	opts     Options
	tools    map[string]*Tool
	toolsMu  sync.RWMutex
	server   *http.Server
	watching bool

	// sessions holds streamable-HTTP MCP client sessions (see streamable.go).
	sessions   map[string]*httpSession
	sessionsMu sync.RWMutex

	// resolver is the shared service schema resolver; it owns registry
	// watching and endpoint parsing so discovery is not duplicated with the
	// HTTP API gateway.
	resolver *schema.Resolver

	// limiters holds per-tool rate limiters (nil if rate limiting is disabled).
	limiters   map[string]*rateLimiter
	limitersMu sync.RWMutex

	// breakers holds per-tool circuit breakers (nil if circuit breaking is disabled).
	breakers   map[string]*circuitBreaker
	breakersMu sync.RWMutex
}

// Tool represents an MCP tool (exposed service endpoint)
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	// Scopes lists the auth scopes required to call this tool.
	// An empty list means no scope restriction (subject to Auth provider).
	Scopes []string `json:"scopes,omitempty"`
	// Payment advertises the x402 payment required to call this tool, so
	// the catalog is shoppable — an agent sees the price before calling.
	// Populated at list time when the gateway has payments enabled; nil
	// means free.
	Payment  *PaymentInfo `json:"payment,omitempty"`
	Service  string       `json:"-"`
	Endpoint string       `json:"-"`
	// Handler is an optional direct handler for framework tools that don't
	// go through RPC. When set, handleCallTool calls this instead of making
	// an RPC request.
	Handler func(input map[string]interface{}) (interface{}, error) `json:"-"`
}

// PaymentInfo advertises, in the tool catalog, the x402 payment required
// to call a tool: how much, in what asset, on which network, and where it
// goes. It lets an agent shop the catalog and choose by price before
// calling.
type PaymentInfo struct {
	Amount  string `json:"amount"` // smallest units (e.g. "10000" = 0.01 USDC)
	Network string `json:"network"`
	Asset   string `json:"asset,omitempty"`
	PayTo   string `json:"payTo"`
}

// paymentFor returns the catalog payment info for a tool, or nil if the
// gateway has no payments configured or the tool is free.
func (s *Server) paymentFor(toolName string) *PaymentInfo {
	if s.opts.Payment == nil {
		return nil
	}
	amount := s.opts.Payment.AmountFor(toolName)
	if amount == "" || amount == "0" {
		return nil
	}
	net := s.opts.Payment.Network
	if net == "" {
		net = "base"
	}
	return &PaymentInfo{
		Amount:  amount,
		Network: net,
		Asset:   s.opts.Payment.Asset,
		PayTo:   s.opts.Payment.PayTo,
	}
}

// NewServer creates an MCP gateway server from opts and starts service
// discovery and registry watching. It does not begin serving; call
// (*Server).Serve to start a transport and (*Server).Stop to shut down
// gracefully. This lets callers (e.g. the micro CLI) orchestrate the MCP
// gateway independently of any other gateway.
func NewServer(opts Options) (*Server, error) {
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
		return nil, fmt.Errorf("registry is required")
	}

	server := &Server{
		opts:     opts,
		tools:    make(map[string]*Tool),
		limiters: make(map[string]*rateLimiter),
		breakers: make(map[string]*circuitBreaker),
	}

	// Discover services and build tool list
	if err := server.discoverServices(); err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	// Watch for service changes
	go server.watchServices()

	return server, nil
}

// Serve starts an MCP gateway with the given options.
// For stdio transport, leave Address empty.
// For SSE transport, set Address (e.g., ":3000").
func Serve(opts Options) error {
	server, err := NewServer(opts)
	if err != nil {
		return err
	}
	return server.Serve()
}

// Serve starts the configured transport and blocks until it stops.
func (s *Server) Serve() error {
	// Start server based on transport
	if s.opts.Address != "" {
		return s.serveHTTP()
	}
	return s.serveStdio()
}

// ListenAndServe is a convenience function that starts an MCP gateway on the given address.
func ListenAndServe(address string, opts Options) error {
	opts.Address = address
	return Serve(opts)
}

// discoverServices builds the tool catalog from the shared schema resolver.
// It refreshes the resolver's registry cache so it works standalone (e.g. in
// tests) as well as when driven by the resolver's watch loop.
func (s *Server) discoverServices() error {
	if s.resolver == nil {
		s.resolver = schema.New(s.opts.Registry)
	}
	if _, err := s.resolver.Refresh(); err != nil {
		return err
	}

	s.toolsMu.Lock()
	defer s.toolsMu.Unlock()

	s.tools = make(map[string]*Tool)

	for _, ep := range s.resolver.Endpoints() {
		inputSchema := make(map[string]any, 2)
		inputSchema["type"] = "object"
		props := make(map[string]any, len(ep.Request))
		for _, f := range ep.Request {
			props[f.Name] = map[string]any{
				"type":        schema.JSONType(f.Type),
				"description": fmt.Sprintf("%s field", f.Name),
			}
		}
		inputSchema["properties"] = props

		tool := &Tool{
			Name:        ep.Name,
			Description: ep.Description,
			InputSchema: inputSchema,
			Service:     ep.Service,
			Endpoint:    ep.Method,
		}

		if len(ep.Scopes) > 0 {
			tool.Scopes = ep.Scopes
		}

		// Gateway-level Scopes override service-level scopes
		if s.opts.Scopes != nil {
			if scopes, ok := s.opts.Scopes[tool.Name]; ok {
				tool.Scopes = scopes
			}
		}

		// Add example from metadata if available
		if ep.Example != "" {
			inputSchema["examples"] = []string{ep.Example}
		}

		s.tools[tool.Name] = tool

		// Create rate limiter for this tool if rate limiting is configured
		if s.opts.RateLimit != nil && s.opts.RateLimit.RequestsPerSecond > 0 {
			s.limitersMu.Lock()
			if s.limiters == nil {
				s.limiters = make(map[string]*rateLimiter)
			}
			if _, exists := s.limiters[tool.Name]; !exists {
				s.limiters[tool.Name] = newRateLimiter(
					s.opts.RateLimit.RequestsPerSecond,
					s.opts.RateLimit.Burst,
				)
			}
			s.limitersMu.Unlock()
		}

		// Create circuit breaker for this tool if configured
		if s.opts.CircuitBreaker != nil {
			s.breakersMu.Lock()
			if s.breakers == nil {
				s.breakers = make(map[string]*circuitBreaker)
			}
			if _, exists := s.breakers[tool.Name]; !exists {
				s.breakers[tool.Name] = newCircuitBreaker(*s.opts.CircuitBreaker)
			}
			s.breakersMu.Unlock()
		}
	}

	if err := s.discoverReflectedGRPC(); err != nil {
		return err
	}

	// Register framework primitives as tools.
	// When Auth is configured, they require micro:admin scope.
	s.registerFrameworkTools()

	s.opts.Logger.Printf("[mcp] Discovered %d tools from %d services (incl. framework)", len(s.tools), len(s.resolver.Services()))
	return nil
}

// registerFrameworkTools adds registry, broker, store, and config as MCP tools.
func (s *Server) registerFrameworkTools() {
	addFramework := func(tool *Tool) {
		// When auth is configured, require micro:admin scope
		if s.opts.Auth != nil {
			tool.Scopes = []string{"micro:admin"}
		}
		s.tools[tool.Name] = tool
		if s.opts.RateLimit != nil && s.opts.RateLimit.RequestsPerSecond > 0 {
			s.limitersMu.Lock()
			if _, exists := s.limiters[tool.Name]; !exists {
				s.limiters[tool.Name] = newRateLimiter(s.opts.RateLimit.RequestsPerSecond, s.opts.RateLimit.Burst)
			}
			s.limitersMu.Unlock()
		}
		if s.opts.CircuitBreaker != nil {
			s.breakersMu.Lock()
			if _, exists := s.breakers[tool.Name]; !exists {
				s.breakers[tool.Name] = newCircuitBreaker(*s.opts.CircuitBreaker)
			}
			s.breakersMu.Unlock()
		}
	}

	addFramework(&Tool{
		Name:        "micro_registry_list",
		Description: "List all registered services in the service registry",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		Handler: func(input map[string]interface{}) (interface{}, error) {
			services, err := s.opts.Registry.ListServices()
			if err != nil {
				return nil, err
			}
			var names []string
			for _, svc := range services {
				names = append(names, svc.Name)
			}
			return map[string]interface{}{"services": names}, nil
		},
	})

	addFramework(&Tool{
		Name:        "micro_registry_get",
		Description: "Get details for a registered service including nodes and endpoints",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "Service name"},
			},
		},
		Handler: func(input map[string]interface{}) (interface{}, error) {
			name, _ := input["name"].(string)
			if name == "" {
				return nil, fmt.Errorf("name is required")
			}
			services, err := s.opts.Registry.GetService(name)
			if err != nil {
				return nil, err
			}
			return services, nil
		},
	})

	addFramework(&Tool{
		Name:        "micro_store_list",
		Description: "List keys in the data store",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		Handler: func(input map[string]interface{}) (interface{}, error) {
			keys, err := store.List()
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"keys": keys}, nil
		},
	})

	addFramework(&Tool{
		Name:        "micro_store_read",
		Description: "Read a record from the data store by key",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{"type": "string", "description": "Record key"},
			},
		},
		Handler: func(input map[string]interface{}) (interface{}, error) {
			key, _ := input["key"].(string)
			if key == "" {
				return nil, fmt.Errorf("key is required")
			}
			records, err := store.Read(key)
			if err != nil {
				return nil, err
			}
			if len(records) == 0 {
				return map[string]interface{}{"error": "not found"}, nil
			}
			return map[string]interface{}{"key": key, "value": string(records[0].Value)}, nil
		},
	})

	addFramework(&Tool{
		Name:        "micro_store_write",
		Description: "Write a record to the data store",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key":   map[string]interface{}{"type": "string", "description": "Record key"},
				"value": map[string]interface{}{"type": "string", "description": "Record value"},
			},
		},
		Handler: func(input map[string]interface{}) (interface{}, error) {
			key, _ := input["key"].(string)
			value, _ := input["value"].(string)
			if key == "" {
				return nil, fmt.Errorf("key is required")
			}
			if err := store.Write(&store.Record{Key: key, Value: []byte(value)}); err != nil {
				return nil, err
			}
			return map[string]interface{}{"status": "ok", "key": key}, nil
		},
	})

	addFramework(&Tool{
		Name:        "micro_broker_publish",
		Description: "Publish a message to a broker topic",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"topic":   map[string]interface{}{"type": "string", "description": "Topic name"},
				"message": map[string]interface{}{"type": "string", "description": "Message body"},
			},
		},
		Handler: func(input map[string]interface{}) (interface{}, error) {
			topic, _ := input["topic"].(string)
			message, _ := input["message"].(string)
			if topic == "" {
				return nil, fmt.Errorf("topic is required")
			}
			b := broker.DefaultBroker
			if err := b.Connect(); err != nil {
				return nil, err
			}
			if err := b.Publish(topic, &broker.Message{Body: []byte(message)}); err != nil {
				return nil, err
			}
			return map[string]interface{}{"status": "ok", "topic": topic}, nil
		},
	})
}

// watchServices watches for service registry changes via the shared schema
// resolver and rebuilds the tool catalog on each change.
func (s *Server) watchServices() {
	if s.watching {
		return
	}
	s.watching = true

	if s.resolver == nil {
		s.resolver = schema.New(s.opts.Registry)
	}
	s.resolver.Start(s.opts.Context)
	for range s.resolver.Changes() {
		// Rediscover services on any change
		if err := s.discoverServices(); err != nil {
			s.opts.Logger.Printf("[mcp] Failed to rediscover services: %v", err)
		}
	}
}

// handler returns the HTTP mux with all MCP routes. Shared by serveHTTP and
// tests.
func (s *Server) handler() *http.ServeMux {
	if s.sessions == nil {
		s.sessions = make(map[string]*httpSession)
	}

	mux := http.NewServeMux()

	// Legacy REST API. Tool calls can be gated behind an x402 payment
	// (enforced per-tool inside invokeTool); listing tools and health stay
	// free.
	if s.opts.Payment != nil {
		net := s.opts.Payment.Network
		if net == "" {
			net = "base"
		}
		s.opts.Logger.Printf("[mcp] x402 payments enabled (network=%s, payTo=%s)", net, s.opts.Payment.PayTo)
	}
	mux.HandleFunc("/mcp/tools", s.handleListTools)
	mux.HandleFunc("/mcp/call", s.handleCallTool)
	mux.HandleFunc("/health", s.handleHealth)

	// Streamable-HTTP MCP transport (JSON-RPC 2.0 over POST/GET/DELETE).
	// This is the endpoint for spec-compliant MCP clients.
	mux.HandleFunc("/mcp", s.handleStreamableHTTP)

	// WebSocket endpoint for bidirectional streaming
	mux.Handle("/mcp/ws", NewWebSocketTransport(s))

	return mux
}

// serveHTTP starts an HTTP server with SSE and WebSocket transports
func (s *Server) serveHTTP() error {
	ctx := s.opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	go s.sweepSessions(ctx)

	s.server = &http.Server{
		Addr:    s.opts.Address,
		Handler: s.handler(),
	}

	// Stop the server when the context is canceled (e.g. Ctrl-C).
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()

	s.opts.Logger.Printf("[mcp] MCP gateway listening on %s (HTTP + WebSocket)", s.opts.Address)
	err := s.server.ListenAndServe()
	if err == http.ErrServerClosed && ctx.Err() != nil {
		return nil
	}
	return err
}

// serveStdio starts stdio-based MCP server (for Claude Code, etc.)
func (s *Server) serveStdio() error {
	transport := NewStdioTransport(s)
	return transport.Serve()
}

// toolCatalog returns a snapshot of the tool catalog, attaching payment info
// for the catalog. Callers must not mutate the returned tools.
func (s *Server) toolCatalog() []*Tool {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()

	tools := make([]*Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		// Attach payment info for the catalog. Copy when pricing so the
		// shared tool struct isn't mutated.
		if pay := s.paymentFor(tool.Name); pay != nil {
			cp := *tool
			cp.Payment = pay
			tools = append(tools, &cp)
			continue
		}
		tools = append(tools, tool)
	}
	return tools
}

// handleListTools returns the list of available tools
func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	if s.opts.AuthFunc != nil {
		if err := s.opts.AuthFunc(r); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": s.toolCatalog(),
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

	payload, traceID, raw, err := s.invokeTool(w, r, req.Tool, req.Input)
	if err == errResponseWritten {
		return
	}
	if err != nil {
		if te, ok := err.(*toolError); ok {
			http.Error(w, te.message, te.status)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if raw {
		// Framework tools respond directly with their result.
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
		return
	}

	// Return response with trace ID
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(TraceIDKey, traceID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"result":   payload,
		"trace_id": traceID,
	})
}

// errResponseWritten marks that the x402 payment gate already wrote an HTTP
// response (the 402 challenge); the caller must not write anything further.
var errResponseWritten = errors.New("x402 response already written")

// toolError is a tool-call failure carrying the HTTP status the legacy REST
// transport returns. The streamable MCP transport maps it to a JSON-RPC error
// (or an isError result for execution failures).
type toolError struct {
	status  int
	message string
}

func (e *toolError) Error() string { return e.message }

// toolCallRequest carries the transport-agnostic inputs for invokeToolCore.
// Each transport (HTTP, stdio, websocket) populates this from its own
// protocol-specific sources.
type toolCallRequest struct {
	ToolName  string
	Input     map[string]interface{}
	Token     string        // bearer token; "" skips auth when Auth is nil
	Account   *auth.Account // pre-authenticated account (skips token inspection)
	BaseCtx   context.Context
	Transport string // "http", "stdio", "websocket", etc.
}

// invokeToolCore runs the shared tool-call pipeline: lookup, auth/scope
// inspection, rate limiting, circuit breaker, tracing, audit, and dispatch
// (RPC or framework handler). It is transport-agnostic — x402 payment
// gating is handled by the HTTP-specific caller (invokeTool) before this.
// raw is true when the tool has a direct Handler (framework tool).
func (s *Server) invokeToolCore(req toolCallRequest) (json.RawMessage, string, bool, error) {
	// Tool lookup
	s.toolsMu.RLock()
	tool, exists := s.tools[req.ToolName]
	s.toolsMu.RUnlock()

	if !exists {
		return nil, "", false, &toolError{status: http.StatusNotFound, message: "Tool not found"}
	}

	// Trace ID + OTel span
	traceID := uuid.New().String()
	ctx, span := s.startToolSpan(req.BaseCtx, req.ToolName, req.Transport, traceID)
	defer span.End()

	// Authenticate and authorize
	var account *auth.Account
	if req.Account != nil {
		// Pre-authenticated account (e.g. websocket connection-level auth).
		account = req.Account
	} else if s.opts.Auth != nil {
		token := strings.TrimPrefix(req.Token, "Bearer ")
		if token == "" {
			span.SetAttributes(attribute.Bool(AttrAuthAllowed, false), attribute.String(AttrAuthDeniedReason, "missing token"))
			setSpanError(span, fmt.Errorf("missing token"))
			s.audit(AuditRecord{TraceID: traceID, Timestamp: time.Now(), Tool: req.ToolName, Allowed: false, DeniedReason: "missing token"})
			return nil, traceID, false, &toolError{status: http.StatusUnauthorized, message: "Unauthorized"}
		}
		acc, err := s.opts.Auth.Inspect(token)
		if err != nil {
			span.SetAttributes(attribute.Bool(AttrAuthAllowed, false), attribute.String(AttrAuthDeniedReason, "invalid token"))
			setSpanError(span, fmt.Errorf("invalid token"))
			s.audit(AuditRecord{TraceID: traceID, Timestamp: time.Now(), Tool: req.ToolName, Allowed: false, DeniedReason: "invalid token"})
			return nil, traceID, false, &toolError{status: http.StatusUnauthorized, message: "Unauthorized"}
		}
		account = acc
	}

	if account != nil {
		span.SetAttributes(attribute.String(AttrAccountID, account.ID))

		// Per-tool scopes
		if len(tool.Scopes) > 0 {
			span.SetAttributes(attribute.StringSlice(AttrScopesRequired, tool.Scopes))
			if !hasScope(account.Scopes, tool.Scopes) {
				span.SetAttributes(attribute.Bool(AttrAuthAllowed, false), attribute.String(AttrAuthDeniedReason, "insufficient scopes"))
				setSpanError(span, fmt.Errorf("insufficient scopes"))
				s.audit(AuditRecord{
					TraceID: traceID, Timestamp: time.Now(), Tool: req.ToolName,
					AccountID: account.ID, ScopesRequired: tool.Scopes,
					Allowed: false, DeniedReason: "insufficient scopes",
				})
				return nil, traceID, false, &toolError{status: http.StatusForbidden, message: "Forbidden: insufficient scopes"}
			}
		}
	}

	// Rate limit
	if err := s.allowRate(req.ToolName); err != nil {
		span.SetAttributes(attribute.Bool(AttrRateLimited, true))
		setSpanError(span, err)
		accountID := ""
		if account != nil {
			accountID = account.ID
		}
		s.audit(AuditRecord{
			TraceID: traceID, Timestamp: time.Now(), Tool: req.ToolName,
			AccountID: accountID, Allowed: false, DeniedReason: "rate limited",
		})
		return nil, traceID, false, &toolError{status: http.StatusTooManyRequests, message: "Rate limit exceeded"}
	}

	span.SetAttributes(attribute.Bool(AttrAuthAllowed, true))

	// Circuit breaker
	if err := s.allowCircuit(req.ToolName); err != nil {
		span.SetAttributes(attribute.String("mcp.circuit_breaker", "open"))
		setSpanError(span, err)
		accountID := ""
		if account != nil {
			accountID = account.ID
		}
		s.audit(AuditRecord{
			TraceID: traceID, Timestamp: time.Now(), Tool: req.ToolName,
			AccountID: accountID, Allowed: false, DeniedReason: "circuit breaker open",
		})
		return nil, traceID, false, &toolError{status: http.StatusServiceUnavailable, message: "Service unavailable: circuit breaker open"}
	}

	// Build context with tracing metadata
	md, _ := metadata.FromContext(ctx)
	if md == nil {
		md = make(metadata.Metadata)
	}
	md.Set(TraceIDKey, traceID)
	md.Set(ToolNameKey, req.ToolName)
	if account != nil {
		md.Set(AccountIDKey, account.ID)
	}
	ctx = metadata.NewContext(ctx, md)

	start := time.Now()

	// Framework tools have a direct handler; service tools go through RPC.
	if tool.Handler != nil {
		result, err := tool.Handler(req.Input)
		if err != nil {
			setSpanError(span, err)
			return nil, traceID, false, &toolError{status: http.StatusInternalServerError, message: err.Error()}
		}
		payload, err := json.Marshal(result)
		if err != nil {
			setSpanError(span, err)
			return nil, traceID, false, &toolError{status: http.StatusInternalServerError, message: err.Error()}
		}
		setSpanOK(span)
		return payload, traceID, true, nil
	}

	// RPC dispatch
	inputBytes, err := json.Marshal(req.Input)
	if err != nil {
		return nil, traceID, false, &toolError{status: http.StatusInternalServerError, message: err.Error()}
	}

	rpcReq := s.opts.Client.NewRequest(tool.Service, tool.Endpoint, &bytes.Frame{Data: inputBytes})
	var rsp bytes.Frame

	if err := s.opts.Client.Call(ctx, rpcReq, &rsp); err != nil {
		s.recordCircuit(req.ToolName, false)
		setSpanError(span, err)
		s.opts.Logger.Printf("[mcp] RPC call failed: %v", err)
		accountID := ""
		if account != nil {
			accountID = account.ID
		}
		s.audit(AuditRecord{
			TraceID: traceID, Timestamp: time.Now(), Tool: req.ToolName,
			AccountID: accountID, ScopesRequired: tool.Scopes,
			Allowed: true, Duration: time.Since(start), Error: err.Error(),
		})
		return nil, traceID, false, &toolError{status: http.StatusInternalServerError, message: fmt.Sprintf("RPC call failed: %v", err)}
	}

	s.recordCircuit(req.ToolName, true)
	setSpanOK(span)

	accountID := ""
	if account != nil {
		accountID = account.ID
	}
	s.audit(AuditRecord{
		TraceID: traceID, Timestamp: time.Now(), Tool: req.ToolName,
		AccountID: accountID, ScopesRequired: tool.Scopes,
		Allowed: true, Duration: time.Since(start),
	})

	return json.RawMessage(rsp.Data), traceID, false, nil
}

// invokeTool runs the HTTP-specific tool-call pipeline: x402 payment gate,
// then delegates to invokeToolCore for the shared auth/rate-limit/circuit-
// breaker/dispatch steps. raw is true for framework tools whose payload is
// the response itself (used by the REST handler to skip the envelope).
func (s *Server) invokeTool(w http.ResponseWriter, r *http.Request, toolName string, input map[string]interface{}) (json.RawMessage, string, bool, error) {
	// x402 payment gate: require the tool's amount before doing work.
	// Free tools (amount "" or "0") pass through; Require writes the 402
	// challenge itself when payment is missing or invalid.
	if s.opts.Payment != nil {
		if !s.opts.Payment.Require(w, r, s.opts.Payment.AmountFor(toolName), toolName) {
			return nil, "", false, errResponseWritten
		}
	}

	// Extract bearer token from HTTP header.
	token := r.Header.Get("Authorization")

	// OTel: continue the caller's trace from HTTP headers.
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	payload, traceID, raw, err := s.invokeToolCore(toolCallRequest{
		ToolName:  toolName,
		Input:     input,
		Token:     token,
		BaseCtx:   ctx,
		Transport: "http",
	})
	if err != nil {
		return nil, traceID, false, err
	}

	return payload, traceID, raw, nil
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

// allowCircuit checks if the tool call is allowed by the circuit breaker.
// Returns nil if allowed, non-nil error if the circuit is open.
func (s *Server) allowCircuit(toolName string) error {
	if s.opts.CircuitBreaker == nil {
		return nil
	}
	s.breakersMu.RLock()
	cb, ok := s.breakers[toolName]
	s.breakersMu.RUnlock()
	if !ok {
		return nil
	}
	return cb.Allow()
}

// recordCircuit records a success or failure for the tool's circuit breaker.
func (s *Server) recordCircuit(toolName string, success bool) {
	if s.opts.CircuitBreaker == nil {
		return
	}
	s.breakersMu.RLock()
	cb, ok := s.breakers[toolName]
	s.breakersMu.RUnlock()
	if !ok {
		return
	}
	if success {
		cb.RecordSuccess()
	} else {
		cb.RecordFailure()
	}
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
		// service := micro.NewService("myservice", )
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

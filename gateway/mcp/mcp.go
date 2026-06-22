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
	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
	"go-micro.dev/v6/wrapper/x402"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
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
		breakers: make(map[string]*circuitBreaker),
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

			// Gateway-level Scopes override service-level scopes
			if s.opts.Scopes != nil {
				if scopes, ok := s.opts.Scopes[toolName]; ok {
					tool.Scopes = scopes
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

			// Create circuit breaker for this tool if configured
			if s.opts.CircuitBreaker != nil {
				s.breakersMu.Lock()
				if _, exists := s.breakers[toolName]; !exists {
					s.breakers[toolName] = newCircuitBreaker(*s.opts.CircuitBreaker)
				}
				s.breakersMu.Unlock()
			}
		}
	}

	// Register framework primitives as tools.
	// When Auth is configured, they require micro:admin scope.
	s.registerFrameworkTools()

	s.opts.Logger.Printf("[mcp] Discovered %d tools from %d services (incl. framework)", len(s.tools), len(services))
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

// serveHTTP starts an HTTP server with SSE and WebSocket transports
func (s *Server) serveHTTP() error {
	mux := http.NewServeMux()

	// MCP endpoints. Tool calls can be gated behind an x402 payment
	// (enforced per-tool inside handleCallTool); listing tools and health
	// stay free.
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

	// WebSocket endpoint for bidirectional streaming
	ws := NewWebSocketTransport(s)
	mux.Handle("/mcp/ws", ws)

	s.server = &http.Server{
		Addr:    s.opts.Address,
		Handler: mux,
	}

	s.opts.Logger.Printf("[mcp] MCP gateway listening on %s (HTTP + WebSocket)", s.opts.Address)
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

	// x402 payment gate: require the tool's amount before doing work.
	// Free tools (amount "" or "0") pass through; Require writes the 402
	// challenge itself when payment is missing or invalid.
	if s.opts.Payment != nil {
		if !s.opts.Payment.Require(w, r, s.opts.Payment.AmountFor(req.Tool), req.Tool) {
			return
		}
	}

	// Generate trace ID for this call
	traceID := uuid.New().String()

	// Start OTel span (noop if TraceProvider is nil)
	ctx, span := s.startToolSpan(r.Context(), req.Tool, "http", traceID)
	defer span.End()

	// Authenticate and authorize
	var account *auth.Account
	if s.opts.Auth != nil {
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")
		if token == "" {
			span.SetAttributes(attribute.Bool(AttrAuthAllowed, false), attribute.String(AttrAuthDeniedReason, "missing token"))
			setSpanError(span, fmt.Errorf("missing token"))
			s.audit(AuditRecord{TraceID: traceID, Timestamp: time.Now(), Tool: req.Tool, Allowed: false, DeniedReason: "missing token"})
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		acc, err := s.opts.Auth.Inspect(token)
		if err != nil {
			span.SetAttributes(attribute.Bool(AttrAuthAllowed, false), attribute.String(AttrAuthDeniedReason, "invalid token"))
			setSpanError(span, fmt.Errorf("invalid token"))
			s.audit(AuditRecord{TraceID: traceID, Timestamp: time.Now(), Tool: req.Tool, Allowed: false, DeniedReason: "invalid token"})
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		account = acc
		span.SetAttributes(attribute.String(AttrAccountID, account.ID))

		// Check per-tool scopes
		if len(tool.Scopes) > 0 {
			span.SetAttributes(attribute.StringSlice(AttrScopesRequired, tool.Scopes))
			if !hasScope(account.Scopes, tool.Scopes) {
				span.SetAttributes(attribute.Bool(AttrAuthAllowed, false), attribute.String(AttrAuthDeniedReason, "insufficient scopes"))
				setSpanError(span, fmt.Errorf("insufficient scopes"))
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
		span.SetAttributes(attribute.Bool(AttrRateLimited, true))
		setSpanError(span, err)
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

	span.SetAttributes(attribute.Bool(AttrAuthAllowed, true))

	// Circuit breaker check
	if err := s.allowCircuit(req.Tool); err != nil {
		span.SetAttributes(attribute.String("mcp.circuit_breaker", "open"))
		setSpanError(span, err)
		accountID := ""
		if account != nil {
			accountID = account.ID
		}
		s.audit(AuditRecord{
			TraceID: traceID, Timestamp: time.Now(), Tool: req.Tool,
			AccountID: accountID, Allowed: false, DeniedReason: "circuit breaker open",
		})
		http.Error(w, "Service unavailable: circuit breaker open", http.StatusServiceUnavailable)
		return
	}

	// Build context with tracing metadata
	// OTel trace context was already injected by startToolSpan; add MCP metadata.
	md, _ := metadata.FromContext(ctx)
	if md == nil {
		md = make(metadata.Metadata)
	}
	md.Set(TraceIDKey, traceID)
	md.Set(ToolNameKey, req.Tool)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
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

	if err := s.opts.Client.Call(ctx, rpcReq, &rsp); err != nil {
		s.recordCircuit(req.Tool, false)
		setSpanError(span, err)
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

	s.recordCircuit(req.Tool, true)
	setSpanOK(span)

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

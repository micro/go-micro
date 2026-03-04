package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/metadata"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/attribute"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocketTransport implements MCP JSON-RPC 2.0 over WebSocket.
// It supports bidirectional streaming for real-time AI agents.
type WebSocketTransport struct {
	server *Server
}

// wsConn wraps a single WebSocket connection with write serialization.
type wsConn struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
	server  *Server
	account *auth.Account // set once during initial auth
}

// NewWebSocketTransport creates a WebSocket transport for the MCP server.
func NewWebSocketTransport(server *Server) *WebSocketTransport {
	return &WebSocketTransport{server: server}
}

// ServeHTTP implements http.Handler and upgrades HTTP to WebSocket.
func (t *WebSocketTransport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		t.server.opts.Logger.Printf("[mcp] WebSocket upgrade failed: %v", err)
		return
	}

	// Extract bearer token from the upgrade request (if present).
	var account *auth.Account
	if t.server.opts.Auth != nil {
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}
		// Allow connection-level auth from header. Per-message auth via
		// _token param is also supported (checked in handleToolsCall).
		if token != "" {
			acc, err := t.server.opts.Auth.Inspect(token)
			if err == nil {
				account = acc
			}
		}
	}

	wc := &wsConn{
		conn:    conn,
		server:  t.server,
		account: account,
	}

	t.server.opts.Logger.Printf("[mcp] WebSocket client connected from %s", r.RemoteAddr)
	go wc.readLoop()
}

// readLoop reads JSON-RPC messages from the WebSocket connection.
func (wc *wsConn) readLoop() {
	defer wc.conn.Close()

	for {
		_, message, err := wc.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				wc.server.opts.Logger.Printf("[mcp] WebSocket read error: %v", err)
			}
			return
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(message, &req); err != nil {
			wc.sendError(nil, ParseError, "Parse error", err.Error())
			continue
		}

		if req.JSONRPC != "2.0" {
			wc.sendError(req.ID, InvalidRequest, "Invalid request", "jsonrpc must be '2.0'")
			continue
		}

		go wc.handleRequest(&req)
	}
}

// handleRequest dispatches a JSON-RPC request to the appropriate handler.
func (wc *wsConn) handleRequest(req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		wc.handleInitialize(req)
	case "tools/list":
		wc.handleToolsList(req)
	case "tools/call":
		wc.handleToolsCall(req)
	default:
		wc.sendError(req.ID, MethodNotFound, "Method not found", req.Method)
	}
}

// handleInitialize handles the initialize request.
func (wc *wsConn) handleInitialize(req *JSONRPCRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "go-micro-mcp",
			"version": "1.0.0",
		},
	}
	wc.sendResponse(req.ID, result)
}

// handleToolsList handles the tools/list request.
func (wc *wsConn) handleToolsList(req *JSONRPCRequest) {
	wc.server.toolsMu.RLock()
	tools := make([]interface{}, 0, len(wc.server.tools))
	for _, tool := range wc.server.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}
	wc.server.toolsMu.RUnlock()

	wc.sendResponse(req.ID, map[string]interface{}{
		"tools": tools,
	})
}

// handleToolsCall handles the tools/call request.
func (wc *wsConn) handleToolsCall(req *JSONRPCRequest) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
		Token     string                 `json:"_token,omitempty"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		wc.sendError(req.ID, InvalidParams, "Invalid params", err.Error())
		return
	}

	// Get tool info
	wc.server.toolsMu.RLock()
	tool, exists := wc.server.tools[params.Name]
	wc.server.toolsMu.RUnlock()

	if !exists {
		wc.sendError(req.ID, InvalidParams, "Tool not found", params.Name)
		return
	}

	traceID := uuid.New().String()

	// Start OTel span
	ctx, span := wc.server.startToolSpan(wc.server.opts.Context, params.Name, "websocket", traceID)
	defer span.End()

	// Resolve account: prefer connection-level auth, fall back to per-message _token.
	account := wc.account
	if wc.server.opts.Auth != nil {
		if account == nil {
			token := params.Token
			if token == "" {
				span.SetAttributes(attribute.Bool(AttrAuthAllowed, false), attribute.String(AttrAuthDeniedReason, "missing token"))
				setSpanError(span, fmt.Errorf("missing token"))
				wc.server.audit(AuditRecord{TraceID: traceID, Timestamp: time.Now(), Tool: params.Name, Allowed: false, DeniedReason: "missing token"})
				wc.sendError(req.ID, InvalidParams, "Unauthorized", "missing token")
				return
			}
			if strings.HasPrefix(token, "Bearer ") {
				token = strings.TrimPrefix(token, "Bearer ")
			}
			acc, err := wc.server.opts.Auth.Inspect(token)
			if err != nil {
				span.SetAttributes(attribute.Bool(AttrAuthAllowed, false), attribute.String(AttrAuthDeniedReason, "invalid token"))
				setSpanError(span, fmt.Errorf("invalid token"))
				wc.server.audit(AuditRecord{TraceID: traceID, Timestamp: time.Now(), Tool: params.Name, Allowed: false, DeniedReason: "invalid token"})
				wc.sendError(req.ID, InvalidParams, "Unauthorized", "invalid token")
				return
			}
			account = acc
		}
		span.SetAttributes(attribute.String(AttrAccountID, account.ID))

		// Check per-tool scopes
		if len(tool.Scopes) > 0 {
			span.SetAttributes(attribute.StringSlice(AttrScopesRequired, tool.Scopes))
			if !hasScope(account.Scopes, tool.Scopes) {
				span.SetAttributes(attribute.Bool(AttrAuthAllowed, false), attribute.String(AttrAuthDeniedReason, "insufficient scopes"))
				setSpanError(span, fmt.Errorf("insufficient scopes"))
				wc.server.audit(AuditRecord{
					TraceID: traceID, Timestamp: time.Now(), Tool: params.Name,
					AccountID: account.ID, ScopesRequired: tool.Scopes,
					Allowed: false, DeniedReason: "insufficient scopes",
				})
				wc.sendError(req.ID, InvalidParams, "Forbidden", "insufficient scopes")
				return
			}
		}
	}

	// Rate limit check
	if err := wc.server.allowRate(params.Name); err != nil {
		span.SetAttributes(attribute.Bool(AttrRateLimited, true))
		setSpanError(span, err)
		accountID := ""
		if account != nil {
			accountID = account.ID
		}
		wc.server.audit(AuditRecord{
			TraceID: traceID, Timestamp: time.Now(), Tool: params.Name,
			AccountID: accountID, Allowed: false, DeniedReason: "rate limited",
		})
		wc.sendError(req.ID, InternalError, "Rate limit exceeded", params.Name)
		return
	}

	span.SetAttributes(attribute.Bool(AttrAuthAllowed, true))

	// Convert arguments to JSON bytes
	inputBytes, err := json.Marshal(params.Arguments)
	if err != nil {
		wc.sendError(req.ID, InternalError, "Failed to marshal arguments", err.Error())
		return
	}

	// Build context with tracing metadata
	md, _ := metadata.FromContext(ctx)
	if md == nil {
		md = make(metadata.Metadata)
	}
	md.Set(TraceIDKey, traceID)
	md.Set(ToolNameKey, params.Name)
	if account != nil {
		md.Set(AccountIDKey, account.ID)
	}
	ctx = metadata.NewContext(ctx, md)

	// Make RPC call
	start := time.Now()
	rpcReq := wc.server.opts.Client.NewRequest(tool.Service, tool.Endpoint, &struct {
		Data []byte
	}{Data: inputBytes})

	var rsp struct {
		Data []byte
	}

	if err := wc.server.opts.Client.Call(ctx, rpcReq, &rsp); err != nil {
		setSpanError(span, err)
		accountID := ""
		if account != nil {
			accountID = account.ID
		}
		wc.server.audit(AuditRecord{
			TraceID: traceID, Timestamp: time.Now(), Tool: params.Name,
			AccountID: accountID, ScopesRequired: tool.Scopes,
			Allowed: true, Duration: time.Since(start), Error: err.Error(),
		})
		wc.sendError(req.ID, InternalError, "RPC call failed", err.Error())
		return
	}

	setSpanOK(span)

	accountID := ""
	if account != nil {
		accountID = account.ID
	}
	wc.server.audit(AuditRecord{
		TraceID: traceID, Timestamp: time.Now(), Tool: params.Name,
		AccountID: accountID, ScopesRequired: tool.Scopes,
		Allowed: true, Duration: time.Since(start),
	})

	// Parse response
	var result interface{}
	if err := json.Unmarshal(rsp.Data, &result); err != nil {
		result = map[string]interface{}{
			"data": string(rsp.Data),
		}
	}

	wc.sendResponse(req.ID, map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("%v", result),
			},
		},
		"trace_id": traceID,
	})
}

// sendResponse sends a JSON-RPC success response.
func (wc *wsConn) sendResponse(id interface{}, result interface{}) {
	wc.writeJSON(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

// sendError sends a JSON-RPC error response.
func (wc *wsConn) sendError(id interface{}, code int, message string, data interface{}) {
	wc.writeJSON(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	})
}

// writeJSON serializes and sends a JSON message over the WebSocket.
func (wc *wsConn) writeJSON(v interface{}) {
	wc.writeMu.Lock()
	defer wc.writeMu.Unlock()

	if err := wc.conn.WriteJSON(v); err != nil {
		wc.server.opts.Logger.Printf("[mcp] WebSocket write error: %v", err)
	}
}

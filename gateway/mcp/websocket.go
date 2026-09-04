package mcp

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"go-micro.dev/v6/auth"

	"github.com/gorilla/websocket"
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
		token = strings.TrimPrefix(token, "Bearer ")
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
		"instructions": mcpInstructions,
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

	// Resolve auth: prefer connection-level account, fall back to per-message _token.
	account := wc.account

	payload, traceID, _, err := wc.server.invokeToolCore(toolCallRequest{
		ToolName:  params.Name,
		Input:     params.Arguments,
		Token:     params.Token,
		Account:   account,
		BaseCtx:   wc.server.opts.Context,
		Transport: "websocket",
	})
	if err != nil {
		if te, ok := err.(*toolError); ok {
			if te.status == http.StatusInternalServerError {
				wc.sendResponse(req.ID, mcpToolError(traceID, te.message))
				return
			}
			// Map HTTP status to JSON-RPC code, preserving original behavior:
			// auth/scope failures → InvalidParams, rate-limit → InternalError.
			code := InternalError
			if te.status == http.StatusUnauthorized || te.status == http.StatusForbidden || te.status == http.StatusNotFound {
				code = InvalidParams
			}
			wc.sendError(req.ID, code, te.message, nil)
			return
		}
		wc.sendError(req.ID, InternalError, "Tool call failed", err.Error())
		return
	}

	wc.sendResponse(req.ID, mcpToolResult(traceID, payload))
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

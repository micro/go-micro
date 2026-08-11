// Streamable-HTTP MCP transport (JSON-RPC 2.0 over POST/GET/DELETE), per the
// MCP spec:
//
//   - POST sends a JSON-RPC request or notification. Requests get an
//     application/json response (or an SSE stream); notifications get HTTP 202.
//   - GET opens a text/event-stream for server→client messages, correlated via
//     the Mcp-Session-Id header.
//   - DELETE terminates a session.
//   - A session is minted on initialize; its id is returned in the
//     Mcp-Session-Id header and must be echoed on later requests.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// mcpSessionHeader is the HTTP header carrying the MCP session id.
const mcpSessionHeader = "Mcp-Session-Id"

// mcpProtocolVersion is advertised (and echoed from the client's request)
// during initialize.
const mcpProtocolVersion = "2025-06-18"

// httpSession tracks one MCP client session: an id for Mcp-Session-Id
// correlation and, once the client opens the GET stream, the pipe for
// server→client messages.
type httpSession struct {
	id       string
	lastSeen time.Time
	ch       chan []byte

	mu   sync.Mutex
	open bool // whether a GET SSE stream is currently open
}

// session returns the registered session for id, or nil.
func (s *Server) session(id string) *httpSession {
	if id == "" {
		return nil
	}
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()
	return s.sessions[id]
}

// newSession mints and registers a session.
func (s *Server) newSession() *httpSession {
	sess := &httpSession{
		id:       uuid.New().String(),
		lastSeen: time.Now(),
		ch:       make(chan []byte, 32),
	}
	s.sessionsMu.Lock()
	s.sessions[sess.id] = sess
	s.sessionsMu.Unlock()
	return sess
}

// emit sends a server→client JSON-RPC message to the session's SSE stream.
// It reports false if the session has no open stream.
func (s *Server) emit(sess *httpSession, msg []byte) bool {
	sess.mu.Lock()
	open := sess.open
	sess.mu.Unlock()
	if !open {
		return false
	}
	select {
	case sess.ch <- msg:
		return true
	default:
		// Slow stream: drop rather than block the caller.
		return true
	}
}

// sweepSessions periodically drops sessions that have been idle and have no
// open stream. ponytail: one background sweep; per-session TTL timers only if
// session churn ever matters.
func (s *Server) sweepSessions(ctx context.Context) {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			cutoff := time.Now().Add(-30 * time.Minute)
			s.sessionsMu.Lock()
			for id, sess := range s.sessions {
				sess.mu.Lock()
				idle := !sess.open && sess.lastSeen.Before(cutoff)
				sess.mu.Unlock()
				if idle {
					delete(s.sessions, id)
				}
			}
			s.sessionsMu.Unlock()
		}
	}
}

// handleStreamableHTTP routes the MCP streamable-HTTP methods.
func (s *Server) handleStreamableHTTP(w http.ResponseWriter, r *http.Request) {
	// The gateway is often called from browser-based MCP clients; allow
	// cross-origin use even without a proxy in front.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, MCP-Protocol-Version, MCP-Session-Id, MCP-Request-Id, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.handleStreamablePOST(w, r)
	case http.MethodGet:
		s.handleStreamableGET(w, r)
	case http.MethodDelete:
		s.handleStreamableDelete(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleStreamablePOST processes a JSON-RPC request or notification (single or
// batch). Requests get an application/json response; notifications get 202.
func (s *Server) handleStreamablePOST(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.Write(streamRPCError(nil, ParseError, "Parse error", err.Error()))
		return
	}
	body = []byte(strings.TrimSpace(string(body)))

	var messages []json.RawMessage
	if len(body) > 0 && body[0] == '[' {
		if err := json.Unmarshal(body, &messages); err != nil {
			w.Write(streamRPCError(nil, ParseError, "Parse error", err.Error()))
			return
		}
	} else {
		messages = []json.RawMessage{body}
	}

	sess := s.session(r.Header.Get(mcpSessionHeader))
	var responses []json.RawMessage
	for _, raw := range messages {
		resp, abort := s.handleStreamMessage(w, r, raw, &sess)
		if abort {
			return
		}
		if resp != nil {
			responses = append(responses, resp)
		}
	}

	if len(responses) == 0 {
		// Notifications only.
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(responses) == 1 {
		w.Write(responses[0])
		return
	}
	w.Write(streamRPCBatch(responses))
}

// handleStreamMessage dispatches one JSON-RPC message. It returns the response
// to write (nil for notifications) and whether the response was already written
// by the x402 gate (abort).
func (s *Server) handleStreamMessage(w http.ResponseWriter, r *http.Request, raw json.RawMessage, sessPtr **httpSession) (json.RawMessage, bool) {
	var rpc struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(raw, &rpc); err != nil {
		return streamRPCError(nil, ParseError, "Parse error", err.Error()), false
	}
	if rpc.JSONRPC != "2.0" {
		return streamRPCError(rpc.ID, InvalidRequest, "Invalid request", "jsonrpc must be '2.0'"), false
	}

	// Notifications carry no id and get no response.
	if len(rpc.ID) == 0 {
		return nil, false
	}

	switch rpc.Method {
	case "initialize":
		sess := *sessPtr
		if sess == nil {
			sess = s.newSession()
			*sessPtr = sess
		}
		sess.lastSeen = time.Now()
		w.Header().Set(mcpSessionHeader, sess.id)
		return s.mcpInitialize(rpc.Params, rpc.ID), false
	case "ping":
		return streamRPCResult(rpc.ID, map[string]interface{}{}), false
	case "tools/list":
		return s.mcpToolsList(rpc.ID), false
	case "tools/call":
		return s.mcpToolsCall(w, r, rpc.Params, rpc.ID)
	default:
		return streamRPCError(rpc.ID, MethodNotFound, "Method not found", rpc.Method), false
	}
}

// mcpInitialize answers the initialize handshake, minting the session that
// handleStreamMessage set on the response headers.
func (s *Server) mcpInitialize(params json.RawMessage, id json.RawMessage) json.RawMessage {
	version := mcpProtocolVersion
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(params, &p); err == nil && p.ProtocolVersion != "" {
		// Echo the client's requested version: every method we implement is
		// version-agnostic, so this always yields a version the client accepts.
		version = p.ProtocolVersion
	}
	return streamRPCResult(id, map[string]interface{}{
		"protocolVersion": version,
		"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
		"serverInfo":      map[string]interface{}{"name": "go-micro-mcp", "version": "1.0.0"},
	})
}

// mcpToolsList answers tools/list from the shared tool catalog.
func (s *Server) mcpToolsList(id json.RawMessage) json.RawMessage {
	return streamRPCResult(id, map[string]interface{}{"tools": s.toolCatalog()})
}

// mcpToolsCall answers tools/call, sharing the REST endpoint's pipeline via
// invokeTool. Tool-execution failures become isError results per the MCP spec;
// pre-flight failures (auth, rate limit, …) become JSON-RPC errors.
func (s *Server) mcpToolsCall(w http.ResponseWriter, r *http.Request, params json.RawMessage, id json.RawMessage) (json.RawMessage, bool) {
	var p struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.Name == "" {
		return streamRPCError(id, InvalidParams, "Invalid params", "name is required"), false
	}

	payload, traceID, _, err := s.invokeTool(w, r, p.Name, p.Arguments)
	if err == errResponseWritten {
		// The x402 gate wrote its own 402 challenge.
		return nil, true
	}
	if err != nil {
		if te, ok := err.(*toolError); ok {
			if te.status == http.StatusInternalServerError {
				return streamRPCResult(id, mcpToolError(traceID, te.message)), false
			}
			return streamRPCError(id, mcpCodeForStatus(te.status), te.message, nil), false
		}
		return streamRPCError(id, InternalError, "Tool call failed", err.Error()), false
	}

	return streamRPCResult(id, mcpToolResult(traceID, payload)), false
}

// mcpCodeForStatus maps a tool-pipeline HTTP status to an MCP JSON-RPC code.
func mcpCodeForStatus(status int) int {
	switch status {
	case http.StatusUnauthorized:
		return -32001 // Unauthorized
	case http.StatusForbidden:
		return -32002 // Forbidden
	case http.StatusNotFound:
		return InvalidParams // tool not found
	default:
		return InternalError
	}
}

// handleStreamableGET opens the server→client SSE stream for a session. A
// request without a (known) session id gets 405 — the reference MCP SDK treats
// that as "no SSE stream offered" and continues.
func (s *Server) handleStreamableGET(w http.ResponseWriter, r *http.Request) {
	sess := s.session(r.Header.Get(mcpSessionHeader))
	if sess == nil {
		http.Error(w, "session required", http.StatusMethodNotAllowed)
		return
	}

	sess.mu.Lock()
	if sess.open {
		sess.mu.Unlock()
		http.Error(w, "stream already open", http.StatusMethodNotAllowed)
		return
	}
	sess.open = true
	sess.lastSeen = time.Now()
	sess.mu.Unlock()

	flusher, ok := w.(http.Flusher)
	if !ok {
		sess.mu.Lock()
		sess.open = false
		sess.mu.Unlock()
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush() // send headers now; the client's GET must not block on the first event

	// Release the session when the client disconnects.
	go func() {
		<-r.Context().Done()
		sess.mu.Lock()
		sess.open = false
		sess.mu.Unlock()
	}()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case msg := <-sess.ch:
			if _, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// handleStreamableDelete terminates a session. The MCP SDK client treats 405 as
// "termination unsupported", so unknown sessions get 405 rather than an error.
func (s *Server) handleStreamableDelete(w http.ResponseWriter, r *http.Request) {
	sid := r.Header.Get(mcpSessionHeader)
	if sid == "" {
		http.Error(w, "session required", http.StatusMethodNotAllowed)
		return
	}
	s.sessionsMu.Lock()
	_, ok := s.sessions[sid]
	delete(s.sessions, sid)
	s.sessionsMu.Unlock()
	if !ok {
		http.Error(w, "unknown session", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func streamRPCResult(id json.RawMessage, result interface{}) json.RawMessage {
	b, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": rawOrNull(id), "result": result})
	return b
}

func streamRPCError(id json.RawMessage, code int, msg string, data interface{}) json.RawMessage {
	b, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      rawOrNull(id),
		"error":   map[string]interface{}{"code": code, "message": msg, "data": data},
	})
	return b
}

func streamRPCBatch(responses []json.RawMessage) json.RawMessage {
	b, _ := json.Marshal(responses)
	return b
}

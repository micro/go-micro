package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// StdioTransport implements MCP JSON-RPC 2.0 over stdio
// This is used by Claude Code and other local AI tools
type StdioTransport struct {
	server   *Server
	reader   *bufio.Reader
	writer   *bufio.Writer
	writerMu sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// NewStdioTransport creates a new stdio transport for the MCP server
func NewStdioTransport(server *Server) *StdioTransport {
	ctx, cancel := context.WithCancel(context.Background())
	return &StdioTransport{
		server: server,
		reader: bufio.NewReader(os.Stdin),
		writer: bufio.NewWriter(os.Stdout),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Serve starts the stdio transport and processes JSON-RPC requests
func (t *StdioTransport) Serve() error {
	t.server.opts.Logger.Printf("[mcp] MCP server started (stdio transport)")

	// Read and process requests from stdin
	for {
		select {
		case <-t.ctx.Done():
			return nil
		default:
		}

		// Read one line (JSON-RPC request)
		line, err := t.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read request: %w", err)
		}

		// Parse JSON-RPC request
		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			t.sendError(nil, ParseError, "Parse error", err.Error())
			continue
		}

		// Validate JSON-RPC version
		if req.JSONRPC != "2.0" {
			t.sendError(req.ID, InvalidRequest, "Invalid request", "jsonrpc must be '2.0'")
			continue
		}

		// Handle request
		go t.handleRequest(&req)
	}
}

// handleRequest processes a single JSON-RPC request
func (t *StdioTransport) handleRequest(req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		t.handleInitialize(req)
	case "tools/list":
		t.handleToolsList(req)
	case "tools/call":
		t.handleToolsCall(req)
	default:
		t.sendError(req.ID, MethodNotFound, "Method not found", req.Method)
	}
}

// handleInitialize handles the initialize request
func (t *StdioTransport) handleInitialize(req *JSONRPCRequest) {
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

	t.sendResponse(req.ID, result)
}

// handleToolsList handles the tools/list request
func (t *StdioTransport) handleToolsList(req *JSONRPCRequest) {
	t.server.toolsMu.RLock()
	tools := make([]interface{}, 0, len(t.server.tools))
	for _, tool := range t.server.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}
	t.server.toolsMu.RUnlock()

	result := map[string]interface{}{
		"tools": tools,
	}

	t.sendResponse(req.ID, result)
}

// handleToolsCall handles the tools/call request
func (t *StdioTransport) handleToolsCall(req *JSONRPCRequest) {
	// Parse params
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		t.sendError(req.ID, InvalidParams, "Invalid params", err.Error())
		return
	}

	// Get tool info
	t.server.toolsMu.RLock()
	tool, exists := t.server.tools[params.Name]
	t.server.toolsMu.RUnlock()

	if !exists {
		t.sendError(req.ID, InvalidParams, "Tool not found", params.Name)
		return
	}

	// Convert arguments to JSON bytes for RPC call
	inputBytes, err := json.Marshal(params.Arguments)
	if err != nil {
		t.sendError(req.ID, InternalError, "Failed to marshal arguments", err.Error())
		return
	}

	// Make RPC call
	rpcReq := t.server.opts.Client.NewRequest(tool.Service, tool.Endpoint, &struct {
		Data []byte
	}{Data: inputBytes})

	var rsp struct {
		Data []byte
	}

	if err := t.server.opts.Client.Call(t.ctx, rpcReq, &rsp); err != nil {
		t.sendError(req.ID, InternalError, "RPC call failed", err.Error())
		return
	}

	// Parse response
	var result interface{}
	if err := json.Unmarshal(rsp.Data, &result); err != nil {
		// If unmarshal fails, return raw data
		result = map[string]interface{}{
			"data": string(rsp.Data),
		}
	}

	t.sendResponse(req.ID, map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("%v", result),
			},
		},
	})
}

// sendResponse sends a JSON-RPC response
func (t *StdioTransport) sendResponse(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	t.writeJSON(resp)
}

// sendError sends a JSON-RPC error response
func (t *StdioTransport) sendError(id interface{}, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	t.writeJSON(resp)
}

// writeJSON writes a JSON-RPC message to stdout
func (t *StdioTransport) writeJSON(v interface{}) {
	t.writerMu.Lock()
	defer t.writerMu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("[mcp] Failed to marshal response: %v", err)
		return
	}

	if _, err := t.writer.Write(data); err != nil {
		log.Printf("[mcp] Failed to write response: %v", err)
		return
	}

	if _, err := t.writer.Write([]byte("\n")); err != nil {
		log.Printf("[mcp] Failed to write newline: %v", err)
		return
	}

	if err := t.writer.Flush(); err != nil {
		log.Printf("[mcp] Failed to flush writer: %v", err)
	}
}

// Stop gracefully stops the stdio transport
func (t *StdioTransport) Stop() error {
	t.cancel()
	return nil
}

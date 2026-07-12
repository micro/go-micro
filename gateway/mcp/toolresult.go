package mcp

// MCP tools/call result shaping, shared by the stdio and websocket JSON-RPC
// transports. Kept in one place so both transports produce spec-shaped results.

// mcpToolResult builds a successful MCP tools/call result. The downstream RPC
// response body (data) is JSON, so it is returned as JSON text — NOT
// fmt.Sprintf("%v", ...) of a decoded value, which produces Go map-syntax
// (map[id:1 name:bob]) instead of JSON and is what an external MCP client
// (e.g. Claude Desktop over stdio) would otherwise receive.
func mcpToolResult(traceID string, data []byte) map[string]interface{} {
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": string(data)},
		},
		"trace_id": traceID,
	}
}

// mcpToolError builds an MCP tools/call result for a tool-EXECUTION failure.
// Per the MCP spec a tool that fails returns a normal result with isError:true
// (the error as text content), NOT a JSON-RPC protocol error — that way the
// agent can read the failure instead of seeing a transport-level error.
func mcpToolError(traceID, msg string) map[string]interface{} {
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": msg},
		},
		"isError":  true,
		"trace_id": traceID,
	}
}

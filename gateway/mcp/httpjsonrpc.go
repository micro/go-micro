package mcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

// HandlerOption configures NewHandler.
type HandlerOption func(*handlerOptions)

type handlerOptions struct {
	serverName, serverVersion, protocolVersion string
}

// WithServerInfo sets the name/version advertised in the initialize response.
func WithServerInfo(name, version string) HandlerOption {
	return func(o *handlerOptions) { o.serverName, o.serverVersion = name, version }
}

// WithProtocolVersion sets the MCP protocol version advertised in initialize.
func WithProtocolVersion(v string) HandlerOption {
	return func(o *handlerOptions) { o.protocolVersion = v }
}

// NewHandler returns an http.Handler serving the MCP protocol over HTTP as
// JSON-RPC 2.0 (initialize, ping, notifications/*, tools/list, tools/call),
// backed by the resolver. Mount it on your own server (e.g. POST /mcp): the
// gateway provides the protocol; you keep your routes, middleware and any
// human-facing docs page.
func NewHandler(r Resolver, opts ...HandlerOption) http.Handler {
	o := handlerOptions{serverName: "go-micro-mcp", serverVersion: "1.0.0", protocolVersion: "2024-11-05"}
	for _, fn := range opts {
		fn(&o)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var rpc struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(req.Body).Decode(&rpc); err != nil {
			writeRPCError(w, nil, ParseError, "Parse error", err.Error())
			return
		}

		// Notifications (and any id-less request) expect no response body.
		if strings.HasPrefix(rpc.Method, "notifications/") || len(rpc.ID) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		ctx := req.Context()
		switch rpc.Method {
		case "initialize":
			writeRPCResult(w, rpc.ID, map[string]interface{}{
				"protocolVersion": o.protocolVersion,
				"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
				"serverInfo":      map[string]interface{}{"name": o.serverName, "version": o.serverVersion},
			})
		case "ping":
			writeRPCResult(w, rpc.ID, map[string]interface{}{})
		case "tools/list":
			tools, err := r.List(ctx)
			if err != nil {
				writeRPCError(w, rpc.ID, InternalError, "Failed to list tools", err.Error())
				return
			}
			list := make([]map[string]interface{}, 0, len(tools))
			for _, t := range tools {
				list = append(list, map[string]interface{}{
					"name": t.Name, "description": t.Description, "inputSchema": t.InputSchema,
				})
			}
			writeRPCResult(w, rpc.ID, map[string]interface{}{"tools": list})
		case "tools/call":
			var p struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			}
			if err := json.Unmarshal(rpc.Params, &p); err != nil {
				writeRPCError(w, rpc.ID, InvalidParams, "Invalid params", err.Error())
				return
			}
			res, err := r.Call(ctx, p.Name, p.Arguments)
			if err != nil {
				// Protocol/pre-check failure -> JSON-RPC error. An *RPCError
				// carries a specific code; anything else is InternalError.
				if rpcErr, ok := err.(*RPCError); ok {
					writeRPCError(w, rpc.ID, rpcErr.Code, rpcErr.Message, rpcErr.Data)
				} else {
					writeRPCError(w, rpc.ID, InternalError, "Tool call failed", err.Error())
				}
				return
			}
			result := map[string]interface{}{
				"content": []map[string]interface{}{{"type": "text", "text": res.Text}},
			}
			if res.IsError {
				result["isError"] = true
			}
			writeRPCResult(w, rpc.ID, result)
		default:
			writeRPCError(w, rpc.ID, MethodNotFound, "Method not found", rpc.Method)
		}
	})
}

func writeRPCResult(w http.ResponseWriter, id json.RawMessage, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": rawOrNull(id), "result": result})
}

func writeRPCError(w http.ResponseWriter, id json.RawMessage, code int, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": rawOrNull(id), "error": map[string]interface{}{"code": code, "message": msg, "data": data}})
}

func rawOrNull(id json.RawMessage) interface{} {
	if len(id) == 0 {
		return nil
	}
	return id
}

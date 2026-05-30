package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"go-micro.dev/v5/client"
	codecBytes "go-micro.dev/v5/codec/bytes"
	"go-micro.dev/v5/registry"
)

type toolNameMap struct {
	mu sync.RWMutex
	m  map[string]string
}

func (n *toolNameMap) put(safe, original string) {
	n.mu.Lock()
	n.m[safe] = original
	n.mu.Unlock()
}

func (n *toolNameMap) get(safe string) (string, bool) {
	n.mu.RLock()
	v, ok := n.m[safe]
	n.mu.RUnlock()
	return v, ok
}

// Tools discovers go-micro services from a registry and converts their
// endpoints into Tool definitions. It also executes tool calls via RPC.
//
// Create with NewTools, discover the tool list with Discover, and wire
// execution into a model with WithTools:
//
//	tools := ai.NewTools(service.Registry())
//	list, _ := tools.Discover()
//	m := ai.New("anthropic", ai.WithAPIKey(key), ai.WithTools(tools))
//	resp, _ := m.Generate(ctx, &ai.Request{Prompt: input, Tools: list})
type Tools struct {
	registry registry.Registry
	client   client.Client
	names    *toolNameMap
}

// ToolOption configures a Tools instance.
type ToolOption func(*Tools)

// ToolClient sets the client used to execute tool calls. Defaults to
// client.DefaultClient.
func ToolClient(c client.Client) ToolOption {
	return func(t *Tools) {
		if c != nil {
			t.client = c
		}
	}
}

// NewTools creates a Tools bound to the given registry.
func NewTools(reg registry.Registry, opts ...ToolOption) *Tools {
	t := &Tools{
		registry: reg,
		client:   client.DefaultClient,
		names:    &toolNameMap{m: map[string]string{}},
	}
	for _, o := range opts {
		o(t)
	}
	return t
}

// Discover walks the registry and returns one Tool per service
// endpoint. Tool names are LLM-safe (dots replaced with underscores).
func (t *Tools) Discover() ([]Tool, error) {
	services, err := t.registry.ListServices()
	if err != nil {
		return nil, err
	}

	var out []Tool
	for _, svc := range services {
		full, err := t.registry.GetService(svc.Name)
		if err != nil || len(full) == 0 {
			continue
		}
		for _, ep := range full[0].Endpoints {
			original := fmt.Sprintf("%s.%s", svc.Name, ep.Name)
			safe := strings.ReplaceAll(original, ".", "_")
			t.names.put(safe, original)

			desc := fmt.Sprintf("Call %s on %s service", ep.Name, svc.Name)
			if ep.Metadata != nil {
				if d, ok := ep.Metadata["description"]; ok && d != "" {
					desc = d
				}
			}

			props := map[string]any{}
			if ep.Request != nil {
				for _, field := range ep.Request.Values {
					props[field.Name] = map[string]any{
						"type":        toolJSONType(field.Type),
						"description": fmt.Sprintf("%s (%s)", field.Name, field.Type),
					}
				}
			}

			out = append(out, Tool{
				Name:         safe,
				OriginalName: original,
				Description:  desc,
				Properties:   props,
			})
		}
	}

	return out, nil
}

// Handler returns a ToolHandler that executes tool calls via RPC using
// the configured client. Tool names may be LLM-safe (underscored) or
// original (dotted). WithTools uses this internally.
func (t *Tools) Handler() ToolHandler {
	c := t.client
	if c == nil {
		c = client.DefaultClient
	}
	return func(name string, input map[string]any) (any, string) {
		if orig, ok := t.names.get(name); ok {
			name = orig
		}
		parts := strings.SplitN(name, ".", 2)
		if len(parts) != 2 {
			return toolErrResult("invalid tool name: " + name)
		}

		inputBytes, err := json.Marshal(input)
		if err != nil {
			return toolErrResult("failed to marshal input: " + err.Error())
		}

		req := c.NewRequest(parts[0], parts[1], &codecBytes.Frame{Data: inputBytes})
		var rsp codecBytes.Frame
		if err := c.Call(context.Background(), req, &rsp); err != nil {
			return toolErrResult(err.Error())
		}

		var result any
		if err := json.Unmarshal(rsp.Data, &result); err != nil {
			result = string(rsp.Data)
		}
		return result, string(rsp.Data)
	}
}

// DiscoverTools is a convenience that discovers tools from a registry
// without creating a Tools instance. For paired discovery + execution,
// create a Tools with NewTools instead.
func DiscoverTools(reg registry.Registry) ([]Tool, error) {
	return NewTools(reg).Discover()
}

func toolErrResult(msg string) (any, string) {
	encoded, _ := json.Marshal(map[string]string{"error": msg})
	return map[string]string{"error": msg}, string(encoded)
}

func toolJSONType(goType string) string {
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

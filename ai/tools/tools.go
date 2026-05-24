// Package tools turns go-micro services into ai.Tool definitions and
// provides an ai.ToolHandler that executes tool calls by issuing RPCs
// to the corresponding service.
//
// This is the building block that lets any go-micro service reason
// about and call other services through an LLM:
//
//	m := ai.New("anthropic",
//	    ai.WithAPIKey(key),
//	    ai.WithToolHandler(tools.Handler(service.Client())),
//	)
//	resp, _ := m.Generate(ctx, &ai.Request{
//	    Prompt: userInput,
//	    Tools:  tools.FromRegistry(service.Registry()),
//	})
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/client"
	codecBytes "go-micro.dev/v5/codec/bytes"
	"go-micro.dev/v5/registry"
)

// nameMap holds the mapping between LLM-safe tool names (no dots) and the
// original `service.Endpoint` names used by the registry/client. Many
// providers reject dots in tool names, so we substitute underscores when
// presenting the tool and restore the original when executing.
type nameMap struct {
	mu sync.RWMutex
	m  map[string]string
}

func (n *nameMap) put(safe, original string) {
	n.mu.Lock()
	n.m[safe] = original
	n.mu.Unlock()
}

func (n *nameMap) get(safe string) (string, bool) {
	n.mu.RLock()
	v, ok := n.m[safe]
	n.mu.RUnlock()
	return v, ok
}

// Set is the shared discovery state between FromRegistry and Handler.
// Use New, then Discover + Handler together when you want the handler
// to recognise LLM-safe tool names that were emitted by Discover.
//
// FromRegistry+Handler are convenience wrappers that create their own
// internal Set; Set is exposed for callers that want to control the
// lifecycle (e.g. cache the tool list and reuse it across turns).
type Set struct {
	registry registry.Registry
	names    *nameMap
}

// New creates an empty Set bound to the given registry. Call Discover
// to populate it. The registry is only used by Discover; Handler does
// not need it.
func New(reg registry.Registry) *Set {
	return &Set{
		registry: reg,
		names:    &nameMap{m: map[string]string{}},
	}
}

// Discover walks the registry and returns one ai.Tool per service
// endpoint. The returned tools have LLM-safe names (dots replaced with
// underscores); the Set remembers the mapping so Handler can route
// calls back to the right service.
func (s *Set) Discover() ([]ai.Tool, error) {
	services, err := s.registry.ListServices()
	if err != nil {
		return nil, err
	}

	var tools []ai.Tool
	for _, svc := range services {
		full, err := s.registry.GetService(svc.Name)
		if err != nil || len(full) == 0 {
			continue
		}
		for _, ep := range full[0].Endpoints {
			original := fmt.Sprintf("%s.%s", svc.Name, ep.Name)
			safe := strings.ReplaceAll(original, ".", "_")
			s.names.put(safe, original)

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
						"type":        jsonType(field.Type),
						"description": fmt.Sprintf("%s (%s)", field.Name, field.Type),
					}
				}
			}

			tools = append(tools, ai.Tool{
				Name:         safe,
				OriginalName: original,
				Description:  desc,
				Properties:   props,
			})
		}
	}

	return tools, nil
}

// Handler returns an ai.ToolHandler that executes tool calls against
// the given client. Tool names may be the LLM-safe form (with
// underscores) emitted by Discover or the original dotted form; both
// resolve to the same RPC.
func (s *Set) Handler(c client.Client) ai.ToolHandler {
	if c == nil {
		c = client.DefaultClient
	}
	return func(name string, input map[string]any) (any, string) {
		if orig, ok := s.names.get(name); ok {
			name = orig
		}
		parts := strings.SplitN(name, ".", 2)
		if len(parts) != 2 {
			return errResult("invalid tool name: " + name)
		}

		inputBytes, err := json.Marshal(input)
		if err != nil {
			return errResult("failed to marshal input: " + err.Error())
		}

		req := c.NewRequest(parts[0], parts[1], &codecBytes.Frame{Data: inputBytes})
		var rsp codecBytes.Frame
		if err := c.Call(context.Background(), req, &rsp); err != nil {
			return errResult(err.Error())
		}

		var result any
		if err := json.Unmarshal(rsp.Data, &result); err != nil {
			result = string(rsp.Data)
		}
		return result, string(rsp.Data)
	}
}

// FromRegistry is a convenience that builds a one-shot Set, discovers
// tools, and returns just the tool list. Use NewSet directly if you
// need to also wire up Handler against the same name mapping.
func FromRegistry(reg registry.Registry) ([]ai.Tool, error) {
	return New(reg).Discover()
}

// Handler is a convenience that returns an ai.ToolHandler bound to the
// given client. It only resolves dotted "service.Endpoint" names — it
// has no awareness of any LLM-safe name mapping. For full round-tripping
// of underscored names emitted by FromRegistry, construct a Set with
// New and call Set.Handler.
func Handler(c client.Client) ai.ToolHandler {
	return (&Set{names: &nameMap{m: map[string]string{}}}).Handler(c)
}

func errResult(msg string) (any, string) {
	encoded, _ := json.Marshal(map[string]string{"error": msg})
	return map[string]string{"error": msg}, string(encoded)
}

// jsonType maps Go types to JSON schema types. Anything that isn't a
// recognised primitive becomes "object".
func jsonType(goType string) string {
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

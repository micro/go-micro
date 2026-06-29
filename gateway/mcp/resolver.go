package mcp

import (
	"context"
	"fmt"
	"sync"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/registry"
)

// ToolFunc executes a manually-registered tool.
type ToolFunc func(ctx context.Context, args map[string]any) (string, error)

// Resolver supplies the gateway's tools and executes calls. Swapping the
// resolver changes where tools come from without touching the MCP protocol or
// transport:
//
//   - NewManualResolver:   tools you register explicitly (full product control,
//     including tools that are not go-micro services, executed via your own
//     logic — auth, metering, …).
//   - NewRegistryResolver: tools auto-discovered from registered services.
//
// The built-in store/broker tools are intentionally NOT exposed by any
// resolver — they remain a development convenience on the legacy Serve() path,
// never a product default.
type Resolver interface {
	// List returns the current tool catalog.
	List(ctx context.Context) ([]Tool, error)
	// Call executes a tool by name with JSON arguments, returning its text result.
	Call(ctx context.Context, name string, args map[string]any) (string, error)
}

// ManualResolver exposes an explicitly-registered set of tools.
type ManualResolver struct {
	mu    sync.RWMutex
	order []Tool
	funcs map[string]ToolFunc
}

// NewManualResolver returns an empty manual resolver.
func NewManualResolver() *ManualResolver {
	return &ManualResolver{funcs: map[string]ToolFunc{}}
}

// Add registers (or replaces) a tool and its handler. Returns the resolver for
// chaining.
func (m *ManualResolver) Add(t Tool, fn ToolFunc) *ManualResolver {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.funcs[t.Name]; ok {
		for i := range m.order {
			if m.order[i].Name == t.Name {
				m.order[i] = t
			}
		}
	} else {
		m.order = append(m.order, t)
	}
	m.funcs[t.Name] = fn
	return m
}

// List returns the registered tools.
func (m *ManualResolver) List(_ context.Context) ([]Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Tool, len(m.order))
	copy(out, m.order)
	return out, nil
}

// Call runs the handler registered for name.
func (m *ManualResolver) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	m.mu.RLock()
	fn, ok := m.funcs[name]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return fn(ctx, args)
}

// RegistryResolver auto-discovers tools from registered go-micro services and
// executes them over RPC. It exposes only services — never the internal
// store/broker tools.
type RegistryResolver struct {
	tools *ai.Tools
}

// NewRegistryResolver discovers services from reg and calls them with cl.
func NewRegistryResolver(reg registry.Registry, cl client.Client) *RegistryResolver {
	return &RegistryResolver{tools: ai.NewTools(reg, ai.ToolClient(cl))}
}

// List discovers the current service tools.
func (r *RegistryResolver) List(_ context.Context) ([]Tool, error) {
	discovered, err := r.tools.Discover()
	if err != nil {
		return nil, err
	}
	out := make([]Tool, 0, len(discovered))
	for _, t := range discovered {
		out = append(out, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: map[string]interface{}{"type": "object", "properties": t.Properties},
		})
	}
	return out, nil
}

// Call executes a discovered service tool.
func (r *RegistryResolver) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	res := r.tools.Handler()(ctx, ai.ToolCall{ID: "1", Name: name, Input: args})
	return res.Content, nil
}

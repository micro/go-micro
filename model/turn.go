package model

import (
	"context"
	"fmt"
)

// Turner performs exactly one provider turn without executing tools. Providers
// retain protocol-specific reasoning and signatures in Continuation.
// Generate remains available for v6 callers during migration.
type Turner interface {
	Turn(context.Context, *Request, ...GenerateOption) (*Response, error)
}

// Continuation is opaque provider history. Callers must return it unchanged to
// the same provider, with the ordered results of that turn's tool calls.
type Continuation struct {
	Model string `json:"model,omitempty"`

	Provider string           `json:"provider"`
	Messages []map[string]any `json:"messages"`
	Results  []ToolResult     `json:"results,omitempty"`
}

// SingleTurn is used by provider Turn implementations, never by tool handlers.
func SingleTurn(ctx context.Context, m Model, req *Request, opts ...GenerateOption) (*Response, error) {
	if req.Continuation != nil && req.Continuation.Provider != m.String() {
		return nil, fmt.Errorf("continuation belongs to provider %q", req.Continuation.Provider)
	}
	if req.Continuation != nil && req.Continuation.Model != "" && req.Continuation.Model != m.Options().Model {
		return nil, fmt.Errorf("continuation model changed; start a new run or migrate it explicitly")
	}
	opts = append(opts, func(o *GenerateOptions) { o.SingleTurn = true })
	resp, err := m.Generate(ctx, req, opts...)
	if resp != nil && resp.Continuation != nil {
		resp.Continuation.Provider = m.String()
		resp.Continuation.Model = m.Options().Model
	}
	return resp, err
}

func IsSingleTurn(opts []GenerateOption) bool {
	var o GenerateOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o.SingleTurn
}

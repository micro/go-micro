// Package minimax implements the MiniMax model provider.
//
// MiniMax offers its flagship MiniMax-M3 model via an OpenAI-compatible
// chat completions endpoint.
//
// Usage:
//
//	import _ "go-micro.dev/v6/ai/minimax"
//
//	m := ai.New("minimax",
//	    ai.WithAPIKey("your-api-key"),
//	)
package minimax

import (
	"context"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/ai/internal/openaiapi"
)

func init() {
	ai.Register("minimax", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
	ai.RegisterStream("minimax")
	ai.RegisterToolStream("minimax")
}

type Provider struct {
	core *openaiapi.Client
}

func NewProvider(opts ...ai.Option) *Provider {
	return &Provider{core: openaiapi.New(openaiapi.Config{
		Name:         "minimax",
		DefaultBase:  "https://api.minimax.io",
		DefaultModel: "MiniMax-M3",
		UseMessages:  true,
	}, opts...)}
}

func (p *Provider) Init(opts ...ai.Option) error { return p.core.Init(opts...) }
func (p *Provider) Options() ai.Options          { return p.core.Options() }
func (p *Provider) String() string               { return p.core.String() }

func (p *Provider) Generate(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (*ai.Response, error) {
	return p.core.Generate(ctx, req, opts...)
}

func (p *Provider) Stream(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (ai.Stream, error) {
	return p.core.Stream(ctx, req, opts...)
}

// Package mistral implements the Mistral AI model provider.
//
// Mistral AI is a European AI company offering high-performance models
// via an OpenAI-compatible chat completions endpoint.
//
// Usage:
//
//	import _ "go-micro.dev/v6/ai/mistral"
//
//	m := ai.New("mistral",
//	    ai.WithAPIKey("your-api-key"),
//	)
package mistral

import (
	"context"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/ai/internal/openaiapi"
)

func init() {
	ai.Register("mistral", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
	ai.RegisterStream("mistral")
	ai.RegisterToolStream("mistral")
}

type Provider struct {
	core *openaiapi.Client
}

func NewProvider(opts ...ai.Option) *Provider {
	return &Provider{core: openaiapi.New(openaiapi.Config{
		Name:         "mistral",
		DefaultBase:  "https://api.mistral.ai",
		DefaultModel: "mistral-large-latest",
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

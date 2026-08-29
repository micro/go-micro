// Package groq implements the Groq model provider.
//
// Groq provides ultra-fast inference for open-weight models via an
// OpenAI-compatible chat completions endpoint.
//
// Usage:
//
//	import _ "go-micro.dev/v6/ai/groq"
//
//	m := ai.New("groq",
//	    ai.WithAPIKey("your-api-key"),
//	)
package groq

import (
	"context"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/ai/internal/openaiapi"
)

func init() {
	ai.Register("groq", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
	ai.RegisterStream("groq")
	ai.RegisterToolStream("groq")
}

type Provider struct {
	core *openaiapi.Client
}

func NewProvider(opts ...ai.Option) *Provider {
	return &Provider{core: openaiapi.New(openaiapi.Config{
		Name:         "groq",
		DefaultBase:  "https://api.groq.com/openai",
		DefaultModel: "llama-3.3-70b-versatile",
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

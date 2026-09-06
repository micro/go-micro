// Package openai implements the OpenAI model provider
package openai

import (
	"context"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/ai/internal/openaiapi"
)

func init() {
	ai.Register("openai", func(opts ...ai.Option) ai.Model {
		return NewProvider(opts...)
	})
	ai.RegisterImage("openai", func(opts ...ai.Option) ai.ImageModel {
		return NewProvider(opts...)
	})
	ai.RegisterStream("openai")
	ai.RegisterToolStream("openai")
}

// Provider implements the ai.Model and ai.ImageModel interfaces for OpenAI.
type Provider struct {
	core *openaiapi.Client
}

// NewProvider creates a new OpenAI provider. It preserves OpenAI's full
// surface: multi-turn messages, max_tokens, reasoning_effort, and token usage.
func NewProvider(opts ...ai.Option) *Provider {
	return &Provider{core: openaiapi.New(openaiapi.Config{
		Name:         "openai",
		DefaultBase:  "https://api.openai.com",
		DefaultModel: "gpt-4o",
		UseMessages:  true,
		ParseUsage:   true,
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

func (p *Provider) GenerateImage(ctx context.Context, req *ai.ImageRequest, opts ...ai.GenerateOption) (*ai.ImageResponse, error) {
	return p.core.GenerateImage(ctx, req, opts...)
}

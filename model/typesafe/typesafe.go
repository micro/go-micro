// Package typesafe implements the TypeSafe System One decision API.
package typesafe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-micro.dev/v6/model"
)

func init() {
	model.RegisterDecider("typesafe", func(opts ...model.Option) model.Decider {
		return NewProvider(opts...)
	})
}

// Provider evaluates typed questions using Jev.
type Provider struct {
	opts model.Options
}

var _ model.Decider = (*Provider)(nil)

// NewProvider creates a provider using jev-latest by default.
func NewProvider(opts ...model.Option) *Provider {
	options := model.NewOptions(opts...)
	if options.Model == "" {
		options.Model = "jev-latest"
	}
	if options.BaseURL == "" {
		options.BaseURL = "https://api.typesafe.ai"
	}
	return &Provider{opts: options}
}

func (p *Provider) Init(opts ...model.Option) error {
	for _, opt := range opts {
		opt(&p.opts)
	}
	return nil
}

func (p *Provider) Options() model.Options { return p.opts }
func (p *Provider) String() string         { return "typesafe" }

// Decide evaluates all questions in one request. It does not retry automatically;
// callers can inspect model.HTTPError and apply their own retry policy.
func (p *Provider) Decide(ctx context.Context, req *model.DecisionRequest) (*model.DecisionResponse, error) {
	if req == nil || len(req.Questions) == 0 {
		return nil, fmt.Errorf("typesafe: questions are required")
	}
	if p.opts.APIKey == "" {
		return nil, fmt.Errorf("typesafe: API key is required")
	}
	input := *req
	if input.Model == "" {
		input.Model = p.opts.Model
	}
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("typesafe: encode request: %w", err)
	}
	if err := validateRequest(body); err != nil {
		return nil, err
	}
	url := strings.TrimRight(p.opts.BaseURL, "/") + "/v1/systemone"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("typesafe: create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+p.opts.APIKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("typesafe: request failed: %w", err)
	}
	defer response.Body.Close()

	const maxResponse = 16 << 20
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if err != nil {
		return nil, fmt.Errorf("typesafe: read response: %w", err)
	}
	if len(data) > maxResponse {
		return nil, fmt.Errorf("typesafe: response exceeds 16 MiB")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, model.NewHTTPError(response, data)
	}
	return decodeResponse(data, req.Questions)
}

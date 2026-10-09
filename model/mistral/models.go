package mistral

import (
	"context"
	"go-micro.dev/v6/model/internal/catalog"
	"strings"
)

// Models queries the provider's model catalog using its configured endpoint.
func (p *Provider) Models(ctx context.Context) ([]string, error) {
	return catalog.List(ctx, strings.TrimRight(p.opts.BaseURL, "/")+"/v1/models", p.opts.APIKey, "openai")
}

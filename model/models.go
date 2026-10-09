package model

import (
	"context"
	"errors"
	"sort"
)

// ModelLister is optionally implemented by providers with a model catalog.
// IDs are provider-defined; listing does not guarantee access or tool support.
type ModelLister interface {
	Models(context.Context) ([]string, error)
}

var ErrModelsUnsupported = errors.New("model: provider does not support model listing")

// ListModels returns sorted, unique IDs from the provider's live catalog.
func ListModels(ctx context.Context, provider Model) ([]string, error) {
	lister, ok := provider.(ModelLister)
	if !ok {
		return nil, ErrModelsUnsupported
	}
	names, err := lister.Models(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var result []string
	for _, name := range names {
		if name != "" && !seen[name] {
			result = append(result, name)
			seen[name] = true
		}
	}
	sort.Strings(result)
	return result, nil
}

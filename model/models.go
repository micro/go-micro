package model

import (
	"context"
	"errors"
	"sort"
	"strings"
)

// ModelLister is optionally implemented by providers with a model catalog.
// IDs are provider-defined; listing does not guarantee access or tool support.
type ModelLister interface {
	Models(context.Context) ([]string, error)
}

var ErrModelsUnsupported = errors.New("model: provider does not support model listing")

// ListModels returns unique choices from the provider's live catalog, sorted
// with the current MiniMax text models in recommended order. Explicit model IDs
// remain usable independently of this list.
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
		lower := strings.ToLower(name)
		prefix := "minimax/"
		if provider.String() == "together" {
			prefix = "minimaxai/"
		}
		family := strings.TrimPrefix(lower, prefix+"minimax-m")
		miniMaxText := (family != lower && len(family) > 0 && family[0] >= '0' && family[0] <= '9') || strings.HasPrefix(lower, prefix+"minimax-text-") || lower == prefix+"minimax-01"
		if miniMaxText {
			switch provider.String() {
			case "openrouter":
				if name != "minimax/minimax-m3.1-flash:free" && name != "minimax/minimax-m3" {
					continue
				}
			case "together":
				if name != "MiniMaxAI/MiniMax-M3" {
					continue
				}
			}
		}
		if name != "" && !seen[name] {
			result = append(result, name)
			seen[name] = true
		}
	}
	sort.Strings(result)
	if provider.String() == "openrouter" {
		m3, flash := -1, -1
		for i, name := range result {
			switch name {
			case "minimax/minimax-m3":
				m3 = i
			case "minimax/minimax-m3.1-flash:free":
				flash = i
			}
		}
		if m3 >= 0 && flash > m3 {
			result[m3], result[flash] = result[flash], result[m3]
		}
	}
	return result, nil
}

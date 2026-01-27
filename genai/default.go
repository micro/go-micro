package genai

import (
	"context"
	"sync"
)

var (
	DefaultGenAI GenAI = &noopGenAI{}
	defaultOnce  sync.Once
)

// SetDefault sets the default GenAI provider (can only be called once).
func SetDefault(g GenAI) {
	defaultOnce.Do(func() {
		DefaultGenAI = g
	})
}

// noopGenAI is a no-op implementation that returns errors.
type noopGenAI struct{}

func (n *noopGenAI) Generate(ctx context.Context, prompt string, opts ...Option) (*Result, error) {
	return nil, ErrNoProvider
}

func (n *noopGenAI) Stream(ctx context.Context, prompt string, opts ...Option) (*Stream, error) {
	return nil, ErrNoProvider
}

func (n *noopGenAI) String() string {
	return "noop"
}

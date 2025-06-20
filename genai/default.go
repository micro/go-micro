package genai

import (
	"sync"
)

var (
	DefaultGenAI GenAI = &noopGenAI{}
	defaultOnce  sync.Once
)

func SetDefault(g GenAI) {
	defaultOnce.Do(func() {
		DefaultGenAI = g
	})
}

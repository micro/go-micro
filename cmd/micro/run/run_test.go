package run

import (
	"os"
	"testing"
)

func TestConsoleUnknownProvider(t *testing.T) {
	t.Setenv("MICRO_AI_PROVIDER", "misspelled-provider")
	t.Setenv("MICRO_AI_API_KEY", "test-key")
	// An invalid provider must return before discovery or interactive input.
	runConsole(make(chan os.Signal))
}

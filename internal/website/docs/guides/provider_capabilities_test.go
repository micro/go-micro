package guides_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go-micro.dev/v6/ai"
	_ "go-micro.dev/v6/ai/anthropic"
	_ "go-micro.dev/v6/ai/atlascloud"
	_ "go-micro.dev/v6/ai/gemini"
	_ "go-micro.dev/v6/ai/groq"
	_ "go-micro.dev/v6/ai/mistral"
	_ "go-micro.dev/v6/ai/openai"
	_ "go-micro.dev/v6/ai/together"
)

func TestAIProviderGuideCapabilityMatrixMatchesRegistry(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	guidePath := filepath.Join(filepath.Dir(filename), "ai-provider-guide.md")
	b, err := os.ReadFile(guidePath)
	if err != nil {
		t.Fatalf("read AI provider guide: %v", err)
	}
	guide := string(b)

	for _, row := range ai.CapabilityRows() {
		want := fmt.Sprintf("| `%s` | %s | %s | %s | %s |", row.Provider, yesNo(row.Model), yesNo(row.Image), yesNo(row.Video), yesNo(row.Stream))
		if !strings.Contains(guide, want) {
			t.Fatalf("AI provider guide capability matrix is stale; missing row %q", want)
		}
	}
}

func yesNo(ok bool) string {
	if ok {
		return "Yes"
	}
	return "No"
}

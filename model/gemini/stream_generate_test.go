package gemini

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadContentKeepsSignaturesAndOnlyEmitsVisibleText(t *testing.T) {
	data := `data: {"candidates":[{"content":{"parts":[{"text":"private","thought":true}]}}]}
data: {"candidates":[{"content":{"parts":[{"text":"hello\n"}]}}]}
data: {"candidates":[{"content":{"parts":[{"functionCall":{"name":"echo","args":{"text":"hi"}},"thoughtSignature":"signed"}]},"finishReason":"STOP"}]}
`
	var text strings.Builder
	raw, err := readContent(strings.NewReader(data), func(s string) { text.WriteString(s) })
	if err != nil {
		t.Fatal(err)
	}
	if text.String() != "hello\n" {
		t.Fatalf("visible text=%q", text.String())
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"thoughtSignature":"signed"`) {
		t.Fatalf("lost signature: %s", raw)
	}
	if _, err := readContent(strings.NewReader("data: {}\n"), func(string) {}); err == nil {
		t.Fatal("accepted truncated response")
	}
}

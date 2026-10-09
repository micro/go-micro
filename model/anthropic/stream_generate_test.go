package anthropic

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadMessageKeepsToolInputAndThinkingSignature(t *testing.T) {
	data := `data: {"type":"message_start","message":{"role":"assistant"}}
data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"","signature":""}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"private"}}
data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"signed"}}
data: {"type":"content_block_stop","index":0}
data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}
data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"hello\n"}}
data: {"type":"content_block_stop","index":1}
data: {"type":"content_block_start","index":2,"content_block":{"type":"tool_use","id":"one","name":"echo","input":{}}}
data: {"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"{\"text\":\""}}
data: {"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"hi\"}"}}
data: {"type":"content_block_stop","index":2}
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"}}
data: {"type":"message_stop"}
`
	var text strings.Builder
	raw, err := readMessage(strings.NewReader(data), func(s string) { text.WriteString(s) })
	if err != nil {
		t.Fatal(err)
	}
	if text.String() != "hello\n" {
		t.Fatalf("visible text=%q", text.String())
	}
	var result struct {
		Content []map[string]any `json:"content"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.Content[0]["signature"] != "signed" || result.Content[2]["input"].(map[string]any)["text"] != "hi" {
		t.Fatalf("content=%v", result.Content)
	}
	if _, err := readMessage(strings.NewReader(strings.ReplaceAll(data, "data: {\"type\":\"content_block_stop\",\"index\":2}\n", "")), func(string) {}); err == nil {
		t.Fatal("accepted unfinished tool input")
	}
}

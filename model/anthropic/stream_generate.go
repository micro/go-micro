package anthropic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// readMessage preserves complete content blocks, including thinking signatures,
// for tool follow-ups while publishing text deltas as they arrive.
func readMessage(reader io.Reader, onToken func(string)) ([]byte, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	message := map[string]any{}
	blocks := []map[string]any{}
	inputs := map[int]string{}
	stopped := false
	open := -1
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var event struct {
			Type    string         `json:"type"`
			Index   int            `json:"index"`
			Message map[string]any `json:"message"`
			Block   map[string]any `json:"content_block"`
			Delta   map[string]any `json:"delta"`
			Usage   map[string]any `json:"usage"`
			Error   struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &event); err != nil {
			return nil, err
		}
		switch event.Type {
		case "error":
			return nil, fmt.Errorf("stream: %s", event.Error.Message)
		case "message_start":
			if event.Message == nil {
				return nil, fmt.Errorf("missing stream message")
			}
			message = event.Message
		case "content_block_start":
			if open != -1 || event.Index != len(blocks) || event.Block == nil {
				return nil, fmt.Errorf("invalid content block index %d", event.Index)
			}
			blocks = append(blocks, event.Block)
			open = event.Index
		case "content_block_delta":
			if event.Index != open || event.Index < 0 || event.Index >= len(blocks) {
				return nil, fmt.Errorf("invalid content block index %d", event.Index)
			}
			block := blocks[event.Index]
			kind, _ := event.Delta["type"].(string)
			switch kind {
			case "text_delta", "thinking_delta", "signature_delta":
				key := strings.TrimSuffix(kind, "_delta")
				delta, _ := event.Delta[key].(string)
				previous, _ := block[key].(string)
				block[key] = previous + delta
				if kind == "text_delta" {
					onToken(delta)
				}
			case "input_json_delta":
				delta, _ := event.Delta["partial_json"].(string)
				inputs[event.Index] += delta
			}
		case "content_block_stop":
			if event.Index != open || open < 0 {
				return nil, fmt.Errorf("invalid content block stop")
			}
			open = -1
			if raw, ok := inputs[event.Index]; ok {
				var input map[string]any
				if err := json.Unmarshal([]byte(raw), &input); err != nil {
					return nil, fmt.Errorf("tool arguments: %w", err)
				}
				blocks[event.Index]["input"] = input
			}
		case "message_delta":
			for key, value := range event.Delta {
				message[key] = value
			}
			usage, _ := message["usage"].(map[string]any)
			if usage == nil {
				usage = map[string]any{}
			}
			for key, value := range event.Usage {
				usage[key] = value
			}
			message["usage"] = usage
		case "message_stop":
			stopped = true
		}
		if stopped {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !stopped || open != -1 || message["stop_reason"] == nil {
		return nil, io.ErrUnexpectedEOF
	}
	message["content"] = blocks
	return json.Marshal(message)
}

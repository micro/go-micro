package gemini

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func readContent(reader io.Reader, onToken func(string)) ([]byte, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	var parts []map[string]any
	var finish string
	var usage map[string]any
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var chunk struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
			Candidates []struct {
				Content struct {
					Parts []map[string]any `json:"parts"`
				} `json:"content"`
				Finish string `json:"finishReason"`
			} `json:"candidates"`
			Usage map[string]any `json:"usageMetadata"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &chunk); err != nil {
			return nil, err
		}
		if chunk.Error != nil {
			return nil, fmt.Errorf("stream: %s", chunk.Error.Message)
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
		if len(chunk.Candidates) == 0 {
			continue
		}
		candidate := chunk.Candidates[0]
		for _, part := range candidate.Content.Parts {
			parts = append(parts, part)
			if text, ok := part["text"].(string); ok && part["thought"] != true {
				onToken(text)
			}
		}
		if candidate.Finish != "" {
			finish = candidate.Finish
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if finish == "" {
		return nil, io.ErrUnexpectedEOF
	}
	return json.Marshal(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": parts}, "finishReason": finish}}, "usageMetadata": usage})
}

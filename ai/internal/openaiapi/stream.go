package openaiapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-micro.dev/v6/ai"
)

// Stream opens an OpenAI-compatible chat completions SSE stream.
func Stream(ctx context.Context, opts ai.Options, req *ai.Request, basePath string) (ai.Stream, error) {
	messages := []map[string]any{{"role": "system", "content": req.SystemPrompt}}
	for _, m := range req.Messages {
		messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
	}
	if req.Prompt != "" {
		messages = append(messages, map[string]any{"role": "user", "content": req.Prompt})
	}
	apiReq := map[string]any{
		"model":          opts.Model,
		"messages":       messages,
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	if opts.MaxTokens > 0 {
		apiReq["max_tokens"] = opts.MaxTokens
	}
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stream request: %w", err)
	}
	apiURL := strings.TrimRight(opts.BaseURL, "/") + basePath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+opts.APIKey)

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream API request failed: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("stream API error (%s): %s", httpResp.Status, string(respBody))
	}
	return &StreamReader{body: httpResp.Body, scanner: bufio.NewScanner(httpResp.Body)}, nil
}

// StreamReader reads OpenAI-compatible server-sent event chunks.
type StreamReader struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	closed  bool
}

func (s *StreamReader) Recv() (*ai.Response, error) {
	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil, io.EOF
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("failed to parse stream chunk: %w", err)
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			return &ai.Response{Reply: chunk.Choices[0].Delta.Content}, nil
		}
		if chunk.Usage != nil {
			return &ai.Response{Usage: ai.Usage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
				TotalTokens:  chunk.Usage.TotalTokens,
			}}, nil
		}
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (s *StreamReader) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.body.Close()
}

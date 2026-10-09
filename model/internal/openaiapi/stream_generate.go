package openaiapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"go-micro.dev/v6/model"
)

type streamedTool struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// streamCall assembles one assistant message while delivering text immediately.
// Generate owns tool execution and uses the assembled message for the next turn.
func streamCall(ctx context.Context, opts model.Options, req map[string]any, onToken func(string)) (*model.Response, map[string]any, error) {
	req["stream"] = true
	req["stream_options"] = map[string]any{"include_usage": true}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(opts.BaseURL, "/")+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Authorization", "Bearer "+opts.APIKey)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 65536))
		return nil, nil, model.NewHTTPError(response, data)
	}
	result := &model.Response{}
	calls := map[int]*streamedTool{}
	var text strings.Builder
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	complete := false
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			complete = true
			break
		}
		var chunk struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
			Choices []struct {
				Index        int    `json:"index"`
				FinishReason string `json:"finish_reason"`
				Delta        struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index int `json:"index"`
						streamedTool
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				Input  int `json:"prompt_tokens"`
				Output int `json:"completion_tokens"`
				Total  int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, nil, fmt.Errorf("stream chunk: %w", err)
		}
		if chunk.Error != nil {
			return nil, nil, fmt.Errorf("stream: %s", chunk.Error.Message)
		}
		for _, choice := range chunk.Choices {
			if choice.Index != 0 {
				continue
			}
			if choice.FinishReason != "" {
				result.StopReason = choice.FinishReason
			}
			if choice.Delta.Content != "" {
				text.WriteString(choice.Delta.Content)
				onToken(choice.Delta.Content)
			}
			for _, delta := range choice.Delta.ToolCalls {
				if delta.Index < 0 || delta.Index > 1024 {
					return nil, nil, fmt.Errorf("invalid tool index %d", delta.Index)
				}
				call := calls[delta.Index]
				if call == nil {
					call = &streamedTool{Type: "function"}
					calls[delta.Index] = call
				}
				call.ID += delta.ID
				call.Function.Name += delta.Function.Name
				call.Function.Arguments += delta.Function.Arguments
			}
		}
		if chunk.Usage != nil {
			result.Usage = model.Usage{InputTokens: chunk.Usage.Input, OutputTokens: chunk.Usage.Output, TotalTokens: chunk.Usage.Total}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	if !complete || result.StopReason == "" {
		return nil, nil, io.ErrUnexpectedEOF
	}
	result.Reply = text.String()
	if result.StopReason == "length" && result.Reply == "" && len(calls) == 0 {
		return nil, nil, model.ErrOutputLimit
	}
	indices := make([]int, 0, len(calls))
	for index := range calls {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	raw := make([]*streamedTool, 0, len(calls))
	for _, index := range indices {
		call := calls[index]
		var input map[string]any
		if err := json.Unmarshal([]byte(call.Function.Arguments), &input); err != nil {
			return nil, nil, fmt.Errorf("tool %s arguments: %w", call.Function.Name, err)
		}
		if call.ID == "" || call.Function.Name == "" {
			return nil, nil, fmt.Errorf("incomplete tool call")
		}
		raw = append(raw, call)
		result.ToolCalls = append(result.ToolCalls, model.ToolCall{ID: call.ID, Name: call.Function.Name, Input: input})
	}
	return result, map[string]any{"content": result.Reply, "tool_calls": raw}, nil
}

package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go-micro.dev/v5/genai"
)

const (
	defaultTextModel  = "gpt-4o-mini"
	defaultImageModel = "dall-e-3"
	defaultAudioModel = "tts-1"
	defaultTimeout    = 120 // seconds
)

type openAI struct {
	options genai.Options
	client  *http.Client
}

// New creates a new OpenAI provider.
func New(opts ...genai.Option) genai.GenAI {
	var options genai.Options
	for _, o := range opts {
		o(&options)
	}
	if options.APIKey == "" {
		options.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if options.Timeout == 0 {
		options.Timeout = defaultTimeout
	}

	return &openAI{
		options: options,
		client: &http.Client{
			Timeout: time.Duration(options.Timeout) * time.Second,
		},
	}
}

func (o *openAI) Generate(ctx context.Context, prompt string, opts ...genai.Option) (*genai.Result, error) {
	options := o.options
	for _, opt := range opts {
		opt(&options)
	}

	res := &genai.Result{Prompt: prompt, Type: options.Type}

	var url string
	var body map[string]interface{}

	switch options.Type {
	case "image":
		model := options.Model
		if model == "" {
			model = defaultImageModel
		}
		url = "https://api.openai.com/v1/images/generations"
		body = map[string]interface{}{
			"prompt": prompt,
			"n":      1,
			"size":   "1024x1024",
			"model":  model,
		}
	case "audio":
		model := options.Model
		if model == "" {
			model = defaultAudioModel
		}
		url = "https://api.openai.com/v1/audio/speech"
		body = map[string]interface{}{
			"model": model,
			"input": prompt,
			"voice": "alloy",
		}
	case "text":
		fallthrough
	default:
		model := options.Model
		if model == "" {
			model = defaultTextModel
		}
		url = "https://api.openai.com/v1/chat/completions"
		body = map[string]interface{}{
			"model":    model,
			"messages": []map[string]string{{"role": "user", "content": prompt}},
		}
		if options.MaxTokens > 0 {
			body["max_tokens"] = options.MaxTokens
		}
		if options.Temperature > 0 {
			body["temperature"] = options.Temperature
		}
	}

	// Use custom endpoint if provided
	if options.Endpoint != "" {
		url = options.Endpoint
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+options.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for API errors
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	switch options.Type {
	case "image":
		var result struct {
			Data []struct {
				URL string `json:"url"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		if len(result.Data) == 0 {
			return nil, fmt.Errorf("no image returned")
		}
		res.Text = result.Data[0].URL
		return res, nil

	case "audio":
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read audio data: %w", err)
		}
		res.Data = data
		return res, nil

	default: // text
		var result struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		if len(result.Choices) == 0 {
			return nil, fmt.Errorf("no choices returned")
		}
		res.Text = result.Choices[0].Message.Content
		return res, nil
	}
}

// Stream performs a streaming request for text generation.
func (o *openAI) Stream(ctx context.Context, prompt string, opts ...genai.Option) (*genai.Stream, error) {
	options := o.options
	for _, opt := range opts {
		opt(&options)
	}

	// Only text supports streaming
	if options.Type != "" && options.Type != "text" {
		// For non-text types, fall back to non-streaming
		results := make(chan *genai.Result, 1)
		go func() {
			defer close(results)
			res, err := o.Generate(ctx, prompt, opts...)
			if err != nil {
				results <- &genai.Result{Error: err}
				return
			}
			results <- res
		}()
		return genai.NewStream(results, nil), nil
	}

	model := options.Model
	if model == "" {
		model = defaultTextModel
	}

	body := map[string]interface{}{
		"model":    model,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"stream":   true,
	}
	if options.MaxTokens > 0 {
		body["max_tokens"] = options.MaxTokens
	}
	if options.Temperature > 0 {
		body["temperature"] = options.Temperature
	}

	url := "https://api.openai.com/v1/chat/completions"
	if options.Endpoint != "" {
		url = options.Endpoint
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create cancellable context for the stream
	streamCtx, cancel := context.WithCancel(ctx)

	req, err := http.NewRequestWithContext(streamCtx, "POST", url, bytes.NewReader(b))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+options.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := o.client.Do(req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check for API errors
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		cancel()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	results := make(chan *genai.Result, 16)

	go func() {
		defer close(results)
		defer resp.Body.Close()
		defer cancel()

		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-streamCtx.Done():
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					results <- &genai.Result{Error: err}
				}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue // Skip malformed chunks
			}

			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				select {
				case results <- &genai.Result{
					Prompt: prompt,
					Type:   "text",
					Text:   chunk.Choices[0].Delta.Content,
				}:
				case <-streamCtx.Done():
					return
				}
			}
		}
	}()

	return genai.NewStream(results, cancel), nil
}

func (o *openAI) String() string {
	return "openai"
}

func init() {
	genai.Register("openai", New())
}

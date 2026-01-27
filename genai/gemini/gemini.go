package gemini

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
	defaultModel     = "gemini-2.0-flash"
	defaultEndpoint  = "https://generativelanguage.googleapis.com/v1beta/models/"
	defaultTimeout   = 120 // seconds
)

// gemini implements the GenAI interface using Google Gemini API.
type gemini struct {
	options genai.Options
	client  *http.Client
}

// New creates a new Gemini provider.
func New(opts ...genai.Option) genai.GenAI {
	var options genai.Options
	for _, o := range opts {
		o(&options)
	}
	if options.APIKey == "" {
		options.APIKey = os.Getenv("GEMINI_API_KEY")
	}
	if options.Timeout == 0 {
		options.Timeout = defaultTimeout
	}

	return &gemini{
		options: options,
		client: &http.Client{
			Timeout: time.Duration(options.Timeout) * time.Second,
		},
	}
}

func (g *gemini) Generate(ctx context.Context, prompt string, opts ...genai.Option) (*genai.Result, error) {
	options := g.options
	for _, o := range opts {
		o(&options)
	}

	res := &genai.Result{Prompt: prompt, Type: options.Type}

	endpoint := options.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	model := options.Model
	if model == "" {
		model = defaultModel
	}

	url := endpoint + model + ":generateContent?key=" + options.APIKey

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	}

	// Add generation config if specified
	genConfig := make(map[string]interface{})
	if options.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = options.MaxTokens
	}
	if options.Temperature > 0 {
		genConfig["temperature"] = options.Temperature
	}
	if len(genConfig) > 0 {
		body["generationConfig"] = genConfig
	}

	if options.Type == "audio" {
		body["response_mime_type"] = "audio/wav"
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for API errors
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if options.Type == "audio" {
		var result struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						InlineData struct {
							Data []byte `json:"data"`
						} `json:"inline_data"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
			return nil, fmt.Errorf("no audio returned")
		}
		res.Data = result.Candidates[0].Content.Parts[0].InlineData.Data
		return res, nil
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}
	res.Text = result.Candidates[0].Content.Parts[0].Text
	return res, nil
}

// Stream performs a streaming request.
func (g *gemini) Stream(ctx context.Context, prompt string, opts ...genai.Option) (*genai.Stream, error) {
	options := g.options
	for _, o := range opts {
		o(&options)
	}

	endpoint := options.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	model := options.Model
	if model == "" {
		model = defaultModel
	}

	// Use streaming endpoint
	url := endpoint + model + ":streamGenerateContent?key=" + options.APIKey + "&alt=sse"

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	}

	// Add generation config if specified
	genConfig := make(map[string]interface{})
	if options.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = options.MaxTokens
	}
	if options.Temperature > 0 {
		genConfig["temperature"] = options.Temperature
	}
	if len(genConfig) > 0 {
		body["generationConfig"] = genConfig
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := g.client.Do(req)
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

			var chunk struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}

			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue // Skip malformed chunks
			}

			if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
				text := chunk.Candidates[0].Content.Parts[0].Text
				if text != "" {
					select {
					case results <- &genai.Result{
						Prompt: prompt,
						Type:   "text",
						Text:   text,
					}:
					case <-streamCtx.Done():
						return
					}
				}
			}
		}
	}()

	return genai.NewStream(results, cancel), nil
}

func (g *gemini) String() string {
	return "gemini"
}

func init() {
	genai.Register("gemini", New())
}

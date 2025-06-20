package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"go-micro.dev/v5/genai"
)

// gemini implements the GenAI interface using Google Gemini 2.5 API.
type gemini struct {
	options genai.Options
}

func New(opts ...genai.Option) genai.GenAI {
	var options genai.Options
	for _, o := range opts {
		o(&options)
	}
	if options.APIKey == "" {
		options.APIKey = os.Getenv("GEMINI_API_KEY")
	}
	return &gemini{options: options}
}

func (g *gemini) Generate(prompt string, opts ...genai.Option) (*genai.Result, error) {
	options := g.options
	for _, o := range opts {
		o(&options)
	}
	ctx := context.Background()

	res := &genai.Result{Prompt: prompt, Type: options.Type}

	endpoint := options.Endpoint
	if endpoint == "" {
		endpoint = "https://generativelanguage.googleapis.com/v1beta/models/"
	}

	var url string
	var body map[string]interface{}

	// Determine model to use
	var model string
	switch options.Type {
	case "image":
		if options.Model != "" {
			model = options.Model
		} else {
			model = "gemini-2.5-pro-vision"
		}
		url = endpoint + model + ":generateContent?key=" + options.APIKey
		body = map[string]interface{}{
			"contents": []map[string]interface{}{
				{"parts": []map[string]string{{"text": prompt}}},
			},
		}
	case "audio":
		if options.Model != "" {
			model = options.Model
		} else {
			model = "gemini-2.5-pro"
		}
		url = endpoint + model + ":generateContent?key=" + options.APIKey
		body = map[string]interface{}{
			"contents": []map[string]interface{}{
				{"parts": []map[string]string{{"text": prompt}}},
			},
			"response_mime_type": "audio/wav",
		}
	case "text":
		fallthrough
	default:
		if options.Model != "" {
			model = options.Model
		} else {
			model = "gemini-2.5-pro"
		}
		url = endpoint + model + ":generateContent?key=" + options.APIKey
		body = map[string]interface{}{
			"contents": []map[string]interface{}{
				{"parts": []map[string]string{{"text": prompt}}},
			},
		}
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
			return nil, err
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
		return nil, err
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}
	res.Text = result.Candidates[0].Content.Parts[0].Text
	return res, nil
}

func (g *gemini) Stream(prompt string, opts ...genai.Option) (*genai.Stream, error) {
	results := make(chan *genai.Result)
	go func() {
		defer close(results)
		res, err := g.Generate(prompt, opts...)
		if err != nil {
			// Send error via Stream.Err, not channel
			return
		}
		results <- res
	}()
	return &genai.Stream{Results: results}, nil
}

func init() {
	genai.Register("gemini", New())
}

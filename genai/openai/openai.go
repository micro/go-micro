package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"go-micro.dev/v5/genai"
)

type openAI struct {
	options genai.Options
}

func New(opts ...genai.Option) genai.GenAI {
	var options genai.Options
	for _, o := range opts {
		o(&options)
	}
	if options.APIKey == "" {
		options.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	return &openAI{options: options}
}

func (o *openAI) Generate(prompt string, opts ...genai.Option) (*genai.Result, error) {
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
			model = "dall-e-3"
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
			model = "tts-1"
		}
		url = "https://api.openai.com/v1/audio/speech"
		body = map[string]interface{}{
			"model": model,
			"input": prompt,
			"voice": "alloy", // or another supported voice
		}
	case "text":
		fallthrough
	default:
		model := options.Model
		if model == "" {
			model = "gpt-3.5-turbo"
		}
		url = "https://api.openai.com/v1/chat/completions"
		body = map[string]interface{}{
			"model":    model,
			"messages": []map[string]string{{"role": "user", "content": prompt}},
		}
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+options.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch options.Type {
	case "image":
		var result struct {
			Data []struct {
				URL string `json:"url"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}
		if len(result.Data) == 0 {
			return nil, fmt.Errorf("no image returned")
		}
		res.Text = result.Data[0].URL
		return res, nil
	case "audio":
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		res.Data = data
		return res, nil
	case "text":
		fallthrough
	default:
		var result struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}
		if len(result.Choices) == 0 {
			return nil, fmt.Errorf("no choices returned")
		}
		res.Text = result.Choices[0].Message.Content
		return res, nil
	}
}

func (o *openAI) Stream(prompt string, opts ...genai.Option) (*genai.Stream, error) {
	results := make(chan *genai.Result)
	go func() {
		defer close(results)
		res, err := o.Generate(prompt, opts...)
		if err != nil {
			// Send error via Stream.Err, not channel
			return
		}
		results <- res
	}()
	return &genai.Stream{Results: results}, nil
}

func init() {
	genai.Register("openai", New())
}

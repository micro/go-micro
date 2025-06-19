package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-micro.dev/v5/genai"
	"io"
	"net/http"
)

const (
	openAITextURL   = "https://api.openai.com/v1/chat/completions"
	openAIImageURL  = "https://api.openai.com/v1/images/generations"
	openAISpeechURL = "https://api.openai.com/v1/audio/transcriptions"
)

type openAI struct {
	options genai.Options
}

func New(opts ...genai.Option) genai.GenAI {
	var options genai.Options
	for _, o := range opts {
		o(&options)
	}
	return &openAI{options: options}
}

func (o *openAI) GenerateText(prompt string, opts ...genai.Option) (string, error) {
	options := o.options
	for _, opt := range opts {
		opt(&options)
	}

	body := map[string]interface{}{
		"model":    "gpt-3.5-turbo",
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	}
	b, _ := json.Marshal(body)

	httpReq, err := http.NewRequest("POST", openAITextURL, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+options.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}
	return result.Choices[0].Message.Content, nil
}

func (o *openAI) GenerateImage(prompt string, opts ...genai.Option) (string, error) {
	options := o.options
	for _, opt := range opts {
		opt(&options)
	}

	body := map[string]interface{}{
		"prompt": prompt,
		"n":      1,
		"size":   "512x512",
	}
	b, _ := json.Marshal(body)

	httpReq, err := http.NewRequest("POST", openAIImageURL, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+options.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Data) == 0 {
		return "", fmt.Errorf("no image returned")
	}
	return result.Data[0].URL, nil
}

func (o *openAI) GenerateSpeech(prompt string, opts ...genai.Option) ([]byte, error) {
	options := o.options
	for _, opt := range opts {
		opt(&options)
	}

	body := map[string]interface{}{
		"model": "tts-1",
		"input": prompt,
		"voice": "alloy", // or another supported voice
	}
	b, _ := json.Marshal(body)

	httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/speech", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+options.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func init() {
	genai.Register("openai", New())
}

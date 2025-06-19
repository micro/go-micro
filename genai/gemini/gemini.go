package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"workspaces/go-micro/genai"
)

type gemini struct {
	options genai.Options
}

func New(opts ...genai.Option) genai.GenAI {
	var options genai.Options
	for _, o := range opts {
		o(&options)
	}
	return &gemini{options: options}
}

const (
	geminiTextURL   = "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent"
	geminiImageURL  = "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro-vision:generateContent"
	geminiSpeechURL = "https://speech.googleapis.com/v1/speech:recognize"
)

func (g *gemini) GenerateText(ctx context.Context, req *genai.TextRequest, opts ...genai.Option) (*genai.TextResponse, error) {
	options := g.options
	for _, opt := range opts {
		opt(&options)
	}

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": req.Prompt}}},
		},
	}
	b, _ := json.Marshal(body)

	httpReq, err := http.NewRequest("POST", geminiTextURL+"?key="+options.APIKey, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
	return &genai.TextResponse{Text: result.Candidates[0].Content.Parts[0].Text}, nil
}

func (g *gemini) GenerateImage(ctx context.Context, req *genai.ImageRequest, opts ...genai.Option) (*genai.ImageResponse, error) {
	options := g.options
	for _, opt := range opts {
		opt(&options)
	}

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": req.Prompt}}},
		},
	}
	b, _ := json.Marshal(body)

	httpReq, err := http.NewRequest("POST", geminiImageURL+"?key="+options.APIKey, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					InlineData struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inline_data"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no image returned")
	}
	// Return the base64 image data (could be changed to a URL if needed)
	return &genai.ImageResponse{ImageURL: result.Candidates[0].Content.Parts[0].InlineData.Data}, nil
}

// Gemini does not support speech-to-text. Do not implement SpeechToText.

func init() {
	genai.Register("gemini", New())
}

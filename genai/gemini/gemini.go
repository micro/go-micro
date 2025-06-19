package gemini

import (
	"context"
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

func (g *gemini) GenerateText(ctx context.Context, req *genai.TextRequest, opts ...genai.Option) (*genai.TextResponse, error) {
	// TODO: Implement Gemini text generation
	return &genai.TextResponse{Text: "[Gemini] generated text"}, nil
}

func (g *gemini) GenerateImage(ctx context.Context, req *genai.ImageRequest, opts ...genai.Option) (*genai.ImageResponse, error) {
	// TODO: Implement Gemini image generation
	return &genai.ImageResponse{ImageURL: "https://example.com/gemini-image.png"}, nil
}

func (g *gemini) SpeechToText(ctx context.Context, req *genai.SpeechRequest, opts ...genai.Option) (*genai.SpeechResponse, error) {
	// TODO: Implement Gemini speech-to-text
	return &genai.SpeechResponse{Text: "[Gemini] transcribed text"}, nil
}

func init() {
	genai.Register("gemini", New())
}

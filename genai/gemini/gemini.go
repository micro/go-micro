package gemini

import (
	"context"
	"fmt"
	ggenai "go-micro.dev/v5/genai"
	"google.golang.org/genai"
)

type gemini struct {
	options genai.Options
	client  *ggenai.Client
}

func New(opts ...genai.Option) genai.GenAI {
	var options genai.Options
	for _, o := range opts {
		o(&options)
	}
	client, err := ggenai.NewClient(context.Background(), option.WithAPIKey(options.APIKey))
	if err != nil {
		panic(err) // or handle error appropriately
	}
	return &gemini{options: options, client: client}
}

func (g *gemini) GenerateText(ctx context.Context, req *genai.TextRequest, opts ...genai.Option) (*genai.TextResponse, error) {
	options := g.options
	for _, opt := range opts {
		opt(&options)
	}

	model := "models/gemini-2.5-pro"
	resp, err := g.client.GenerateContent(ctx, &ggenai.GenerateContentRequest{
		Model: model,
		Contents: []*ggenai.Content{{
			Parts: []*ggenai.Part{{
				Data: &genai.Part_Text{Text: req.Prompt},
			}},
		}},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}
	return &genai.TextResponse{Text: resp.Candidates[0].Content.Parts[0].GetText()}, nil
}

func (g *gemini) GenerateImage(ctx context.Context, req *genai.ImageRequest, opts ...genai.Option) (*genai.ImageResponse, error) {
	options := g.options
	for _, opt := range opts {
		opt(&options)
	}

	model := "models/gemini-2.5-pro-vision"
	resp, err := g.client.GenerateContent(ctx, &ggenai.GenerateContentRequest{
		Model: model,
		Contents: []*ggenai.Content{{
			Parts: []*ggenai.Part{{
				Data: &ggenai.Part_Text{Text: req.Prompt},
			}},
		}},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no image returned")
	}
	// GemIni API may return image data as base64 or a URL depending on the model/version
	return &genai.ImageResponse{ImageURL: resp.Candidates[0].Content.Parts[0].GetText()}, nil
}

// Gemini does not support speech-to-text. Do not implement SpeechToText.

func (g *gemini) GenerateSpeech(prompt string, opts ...genai.Option) ([]byte, error) {
	ctx := context.Background()
	model := "models/gemini-2.5-pro"
	resp, err := g.client.GenerateContent(ctx, &genai.GenerateContentRequest{
		Model: model,
		Contents: []*genai.Content{{
			Parts: []*genai.Part{{
				Data: &genai.Part_Text{Text: prompt},
			}},
		}},
		ResponseModality: []genai.ResponseModality{genai.ResponseModality_AUDIO},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, nil
	}
	// The audio data is returned as binary in the part's data
	return resp.Candidates[0].Content.Parts[0].GetAudio(), nil
}

func init() {
	genai.Register("gemini", New())
}

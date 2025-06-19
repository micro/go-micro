package gemini

import (
	"context"
	"fmt"
	"go-micro.dev/v5/genai"
	genaigo "google.golang.org/genai"
)

// gemini implements the GenAI interface using Google Gemini 2.5 API.
type gemini struct {
	options genai.Options
	client  *genaigo.Client
}

func New(opts ...genai.Option) genai.GenAI {
	var options genai.Options
	for _, o := range opts {
		o(&options)
	}
	client, err := genaigo.NewClient(context.Background(), &genaigo.ClientConfig{APIKey: options.APIKey})
	if err != nil {
		panic(err) // or handle error appropriately
	}
	return &gemini{options: options, client: client}
}

func (g *gemini) GenerateText(prompt string, opts ...genai.Option) (string, error) {
	ctx := context.Background()
	resp, err := g.client.Models.GenerateContent(ctx, "gemini-2.5-pro", []*genaigo.Content{{
		Parts: []*genaigo.Part{{
			Text: prompt,
		}},
	}}, nil)
	if err != nil {
		return "", err
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no candidates returned")
	}
	return resp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *gemini) GenerateImage(prompt string, opts ...genai.Option) (string, error) {
	ctx := context.Background()
	resp, err := g.client.Models.GenerateContent(ctx, "gemini-2.5-pro", []*genaigo.Content{{
		Parts: []*genaigo.Part{{
			Text: prompt,
		}},
	}}, nil)
	if err != nil {
		return "", err
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no image returned")
	}
	return resp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *gemini) GenerateSpeech(prompt string, opts ...genai.Option) ([]byte, error) {
	ctx := context.Background()
	resp, err := g.client.Models.GenerateContent(ctx, "gemini-2.5-pro", []*genaigo.Content{{
		Parts: []*genaigo.Part{{
			Text: prompt,
		}},
	}}, &genaigo.GenerateContentConfig{
		ResponseMIMEType: "audio/wav",
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no audio returned")
	}
	return resp.Candidates[0].Content.Parts[0].InlineData.Data, nil
}

func init() {
	genai.Register("gemini", New())
}

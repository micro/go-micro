package ai

import "context"

// ImageModel provides an interface for image generation providers.
// Providers that support image generation implement this alongside
// or instead of Model. Use NewImage to construct, or type-assert
// a provider that implements both:
//
//	p := atlascloud.NewProvider(ai.WithAPIKey(key))
//	if ig, ok := p.(ai.ImageModel); ok {
//	    resp, _ := ig.GenerateImage(ctx, req)
//	}
type ImageModel interface {
	GenerateImage(ctx context.Context, req *ImageRequest, opts ...GenerateOption) (*ImageResponse, error)
	String() string
}

// ImageRequest describes what image to generate.
type ImageRequest struct {
	// Prompt is the text description of the image to generate.
	Prompt string
	// Model overrides the provider's default image model.
	Model string
	// Size of the generated image (e.g. "1024x1024"). Provider-specific.
	Size string
	// N is the number of images to generate. Defaults to 1.
	N int
	// Quality controls generation quality. Provider-specific (e.g. "low", "medium", "high").
	Quality string
	// OutputFormat sets the image format (e.g. "png", "jpeg"). Provider-specific.
	OutputFormat string
}

// ImageResponse holds the generated images.
type ImageResponse struct {
	Images []Image
}

// Image is a single generated image, returned as a URL, base64 data, or both
// depending on the provider and request options.
type Image struct {
	// URL is a remote URL where the image can be fetched.
	URL string
	// Base64 is the base64-encoded image data.
	Base64 string
}

// NewImageFunc creates a new ImageModel instance.
type NewImageFunc func(...Option) ImageModel

var imageProviders = make(map[string]NewImageFunc)

// RegisterImage registers an image generation provider.
func RegisterImage(name string, fn NewImageFunc) {
	imageProviders[name] = fn
}

// NewImage creates a new ImageModel instance based on the provider name.
func NewImage(provider string, opts ...Option) ImageModel {
	if fn, ok := imageProviders[provider]; ok {
		return fn(opts...)
	}
	return nil
}

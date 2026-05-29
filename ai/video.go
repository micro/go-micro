package ai

import "context"

// VideoModel provides an interface for video generation providers.
// Providers that support video generation implement this alongside
// Model and/or ImageModel.
type VideoModel interface {
	GenerateVideo(ctx context.Context, req *VideoRequest, opts ...GenerateOption) (*VideoResponse, error)
	String() string
}

// VideoRequest describes what video to generate.
type VideoRequest struct {
	// Prompt is the text description or instructions for the video.
	Prompt string
	// Model overrides the provider's default video model.
	Model string
	// Images are reference image URLs for image-to-video generation.
	Images []string
	// Duration in seconds. Provider-specific defaults apply.
	Duration int
	// AspectRatio (e.g. "16:9", "9:16"). Provider-specific.
	AspectRatio string
	// Resolution (e.g. "720p", "1080p"). Provider-specific.
	Resolution string
}

// VideoResponse holds the generated video.
type VideoResponse struct {
	// URL is the remote URL where the video can be fetched.
	URL string
}

// NewVideoFunc creates a new VideoModel instance.
type NewVideoFunc func(...Option) VideoModel

var videoProviders = make(map[string]NewVideoFunc)

// RegisterVideo registers a video generation provider.
func RegisterVideo(name string, fn NewVideoFunc) {
	videoProviders[name] = fn
}

// NewVideo creates a new VideoModel instance based on the provider name.
func NewVideo(provider string, opts ...Option) VideoModel {
	if fn, ok := videoProviders[provider]; ok {
		return fn(opts...)
	}
	return nil
}

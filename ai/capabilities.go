package ai

import "sort"

// CapabilityRow is one deterministic row in a provider capability matrix.
type CapabilityRow struct {
	// Provider is the registered provider name.
	Provider string `json:"provider"`
	Capabilities
}

// Capabilities describes the AI interfaces a provider has registered.
// It is intentionally based on package registration rather than external
// provider marketing claims, so it reflects what this build can actually use.
type Capabilities struct {
	// Model reports whether ai.New can construct a chat/text model provider.
	Model bool `json:"model"`
	// Image reports whether ai.NewImage can construct an image model provider.
	Image bool `json:"image"`
	// Video reports whether ai.NewVideo can construct a video model provider.
	Video bool `json:"video"`
	// Stream reports whether the provider has registered end-to-end token streaming.
	// Providers that only satisfy the Model interface with ErrStreamingUnsupported
	// leave this false until their Stream implementation is usable.
	Stream bool `json:"stream"`
}

// ProviderCapabilities reports the capabilities registered for provider.
func ProviderCapabilities(provider string) Capabilities {
	_, hasModel := providers[provider]
	_, hasImage := imageProviders[provider]
	_, hasVideo := videoProviders[provider]
	_, hasStream := streamProviders[provider]

	return Capabilities{
		Model:  hasModel,
		Image:  hasImage,
		Video:  hasVideo,
		Stream: hasStream,
	}
}

// CapabilityMatrix returns a snapshot of all registered AI providers and the
// interfaces they support. The returned map is a copy and can be modified by
// callers without mutating the registry. Use CapabilityRows when rendering a
// deterministic table or report.
func CapabilityMatrix() map[string]Capabilities {
	names := map[string]struct{}{}
	for name := range providers {
		names[name] = struct{}{}
	}
	for name := range imageProviders {
		names[name] = struct{}{}
	}
	for name := range videoProviders {
		names[name] = struct{}{}
	}
	for name := range streamProviders {
		names[name] = struct{}{}
	}

	matrix := make(map[string]Capabilities, len(names))
	for name := range names {
		matrix[name] = ProviderCapabilities(name)
	}
	return matrix
}

// CapabilityRows returns a deterministic capability support matrix for every
// registered AI provider. It is the ordered form of CapabilityMatrix, intended
// for CLIs, docs generators, and conformance reports that need stable output.
func CapabilityRows() []CapabilityRow {
	names := RegisteredProviders("")
	rows := make([]CapabilityRow, 0, len(names))
	for _, name := range names {
		rows = append(rows, CapabilityRow{
			Provider:     name,
			Capabilities: ProviderCapabilities(name),
		})
	}
	return rows
}

// RegisterStream records that provider has a usable Stream implementation.
// Providers should call this from init alongside Register once Stream returns
// chunks instead of ErrStreamingUnsupported.
func RegisterStream(provider string) {
	streamProviders[provider] = struct{}{}
}

var streamProviders = make(map[string]struct{})

// RegisteredProviders returns the registered provider names in sorted order.
// kind may be "model", "image", "video", "stream", or empty for the union of all
// provider registries.
func RegisteredProviders(kind string) []string {
	names := map[string]struct{}{}
	add := func(registry any) {
		switch r := registry.(type) {
		case map[string]NewFunc:
			for name := range r {
				names[name] = struct{}{}
			}
		case map[string]NewImageFunc:
			for name := range r {
				names[name] = struct{}{}
			}
		case map[string]NewVideoFunc:
			for name := range r {
				names[name] = struct{}{}
			}
		case map[string]struct{}:
			for name := range r {
				names[name] = struct{}{}
			}
		}
	}

	switch kind {
	case "model":
		add(providers)
	case "stream":
		add(streamProviders)
	case "image":
		add(imageProviders)
	case "video":
		add(videoProviders)
	default:
		add(providers)
		add(imageProviders)
		add(videoProviders)
	}

	out := make([]string, 0, len(names))
	for name := range names {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

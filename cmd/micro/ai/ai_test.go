package ai

import (
	"bytes"
	"strings"
	"testing"

	goai "go-micro.dev/v6/ai"
)

func TestWriteProviderMatrix(t *testing.T) {
	rows := []goai.CapabilityRow{
		{Provider: "atlascloud", Capabilities: goai.Capabilities{Model: true, Image: true, Video: true}},
		{Provider: "openai", Capabilities: goai.Capabilities{Model: true, Image: true}},
	}

	var out bytes.Buffer
	writeProviderMatrix(&out, rows)
	got := out.String()

	for _, want := range []string{
		"Provider    Model  Image  Video",
		"atlascloud  ✓      ✓      ✓",
		"openai      ✓      ✓      -",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("matrix output missing %q:\n%s", want, got)
		}
	}
}

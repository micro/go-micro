package ai

import (
	"bytes"
	"encoding/json"
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

func TestWriteProviderJSON(t *testing.T) {
	rows := []goai.CapabilityRow{
		{Provider: "openai", Capabilities: goai.Capabilities{Model: true, Image: true}},
	}

	var out bytes.Buffer
	if err := writeProviderJSON(&out, rows); err != nil {
		t.Fatalf("writeProviderJSON returned error: %v", err)
	}

	var got []goai.CapabilityRow
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("JSON output did not decode: %v\n%s", err, out.String())
	}
	if len(got) != 1 || got[0].Provider != "openai" || !got[0].Model || !got[0].Image || got[0].Video {
		t.Fatalf("decoded JSON = %#v, want openai model+image", got)
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Fatalf("JSON output should end with newline: %q", out.String())
	}
}

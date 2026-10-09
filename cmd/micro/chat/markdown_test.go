package chat

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestMarkdownReply(t *testing.T) {
	input := "# Result\n\nA **useful** answer with [docs](https://go-micro.dev).\n\n- first\n- second\n\n~~~go\nif ready {\n    fmt.Println(\"**literal**\")\n}\n~~~\n\n| Name | State |\n| --- | --- |\n| service | ready |"
	output := ansi.Strip(renderMarkdown(input, 70))
	for _, text := range []string{"Result", "useful", "https://go-micro.dev", "first", "second", "**literal**", "    fmt.Println", "service", "ready"} {
		if !strings.Contains(output, text) {
			t.Errorf("missing %q in %q", text, output)
		}
	}
	for _, marker := range []string{"# Result", "**useful**", "~~~go", "| --- |"} {
		if strings.Contains(output, marker) {
			t.Errorf("unrendered markup %q in %q", marker, output)
		}
	}
}

func TestMarkdownNoColorAndPreview(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	rendered := renderMarkdown("# Heading\n\n**Text** and a [link](https://example.com)", 30)
	if strings.Contains(rendered, "\x1b") {
		t.Fatalf("NO_COLOR emitted escapes: %q", rendered)
	}
	preview := markdownPreview("hidden\n短い行\n"+strings.Repeat("wide ", 20), 20, 2)
	if strings.Contains(preview, "hidden") {
		t.Fatalf("preview not bounded: %q", preview)
	}
	for _, line := range strings.Split(preview, "\n") {
		if ansi.StringWidth(line) > 20 {
			t.Fatalf("preview too wide: %q", line)
		}
	}
}

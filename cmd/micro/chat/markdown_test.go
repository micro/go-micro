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

func TestMarkdownNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	rendered := renderMarkdown("# Heading\n\n**Text** and a [link](https://example.com)", 30)
	if strings.Contains(rendered, "\x1b") {
		t.Fatalf("NO_COLOR emitted escapes: %q", rendered)
	}
}

func TestCompleteMarkdown(t *testing.T) {
	for _, test := range []struct{ name, text, ready string }{
		{"partial paragraph", "Hello", ""},
		{"paragraph", "Hello\n\nStill writing", "Hello\n\n"},
		{"open code", "Before\n\n```go\nfirst\n\nsecond", "Before\n\n"},
		{"closed code", "```go\nfirst\n\nsecond\n```\nAfter", "```go\nfirst\n\nsecond\n```\n"},
		{"shorter fence", "````go\n```\n\nmore", ""},
		{"tilde fence", "~~~go\nx\n\n~~~\n", "~~~go\nx\n\n~~~\n"},
		{"partial closing fence", "```go\nx\n```", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.text[:completeMarkdown(test.text)]; got != test.ready {
				t.Fatalf("ready %q, want %q", got, test.ready)
			}
		})
	}
}

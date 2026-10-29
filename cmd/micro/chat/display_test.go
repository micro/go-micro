package chat

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/chzyer/readline"
)

func TestReplyIsPrintedOnce(t *testing.T) {
	var output bytes.Buffer
	terminal, err := readline.NewEx(&readline.Config{
		Prompt: "micro > ", Stdin: io.NopCloser(strings.NewReader("")),
		Stdout: &output, Stderr: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	d := &chatDisplay{terminal: terminal}
	for _, token := range []string{"Please ", "provide ", "an issue number."} {
		d.token(token)
	}
	if strings.Contains(output.String(), "Please") {
		t.Fatalf("partial reply leaked: %q", output.String())
	}
	d.flush()
	if strings.Count(output.String(), "Please") != 1 {
		t.Fatalf("reply not printed once: %q", output.String())
	}
	d.flush()
	if strings.Count(output.String(), "Please") != 1 {
		t.Fatalf("reply repeated: %q", output.String())
	}
}

func TestCompletedBlocksAppearBeforeFlush(t *testing.T) {
	var output bytes.Buffer
	terminal, err := readline.NewEx(&readline.Config{Prompt: "micro > ", Stdin: io.NopCloser(strings.NewReader("")), Stdout: &output, Stderr: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	d := &chatDisplay{terminal: terminal}
	d.token("First paragraph.\n")
	d.token("\n```go\ncode line\n\n")
	if strings.Count(output.String(), "First paragraph.") != 1 {
		t.Fatalf("paragraph not streamed: %q", output.String())
	}
	if strings.Contains(output.String(), "code line") {
		t.Fatal("incomplete code block printed")
	}
	d.token("more code\n```\nLast paragraph.")
	if strings.Count(output.String(), "code line") != 1 {
		t.Fatal("completed code block not streamed")
	}
	d.flush()
	d.flush()
	for _, text := range []string{"First paragraph.", "code line", "more code", "Last paragraph."} {
		if strings.Count(output.String(), text) != 1 {
			t.Fatalf("missing or duplicated %q: %q", text, output.String())
		}
	}
}

func TestStreamedReferenceLinks(t *testing.T) {
	for _, parts := range [][]string{
		{"[Docs][docs]\n\n", "[docs]: https://example.com\n"},
		{"[docs]: https://example.com\n\n", "[Docs][docs]\n\n"},
		{"[docs]\n\n", "[docs]: https://example.com\n"},
		{"[docs][]\n\n", "[docs]: https://example.com\n"},
	} {
		t.Run(strings.Join(parts, ""), func(t *testing.T) {
			var output bytes.Buffer
			terminal, err := readline.NewEx(&readline.Config{Prompt: "micro > ", Stdin: io.NopCloser(strings.NewReader("")), Stdout: &output, Stderr: io.Discard})
			if err != nil {
				t.Fatal(err)
			}
			defer terminal.Close()
			d := &chatDisplay{terminal: terminal}
			d.token("Introduction.\n\n")
			if strings.Count(output.String(), "Introduction.") != 1 {
				t.Fatal("ordinary paragraph did not stream")
			}
			before := output.String()
			for _, part := range parts {
				d.token(part)
			}
			if output.String() != before {
				t.Fatalf("reference context rendered early: %q", output.String())
			}
			d.flush()
			d.flush()
			if strings.Count(output.String(), "https://example.com") != 1 {
				t.Fatalf("link lost or repeated: %q", output.String())
			}
			if strings.Count(output.String(), "Introduction.") != 1 {
				t.Fatal("earlier paragraph repeated")
			}
		})
	}
}

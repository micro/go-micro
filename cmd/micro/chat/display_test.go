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

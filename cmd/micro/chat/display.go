package chat

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode"

	"github.com/chzyer/readline"
)

// chatDisplay keeps reply text out of the editable prompt. Completed Markdown
// blocks are printed once as they arrive. Tool output stays plain text.
type chatDisplay struct {
	terminal *readline.Instance
	mu       sync.Mutex
	pending  string
	markdown strings.Builder
	busy     bool
	name     string
}

func terminalText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s)
}

func (d *chatDisplay) token(text string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.markdown.WriteString(terminalText(text))
	text = d.markdown.String()
	if end := completeMarkdown(text); end > 0 {
		width := d.width()
		fmt.Fprintln(d.terminal.Stdout(), renderMarkdown(text[:end], width))
		d.markdown.Reset()
		d.markdown.WriteString(text[end:])
	}
}

func (d *chatDisplay) width() int {
	width, _, err := readline.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 10 {
		return 80
	}
	return width - 1
}

// output preserves command output, including Markdown-like syntax.
func (d *chatDisplay) output(text string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pending += terminalText(text)
	if index := strings.LastIndexByte(d.pending, '\n'); index >= 0 {
		d.terminal.SetPrompt(d.prompt())
		fmt.Fprint(d.terminal.Stdout(), d.pending[:index+1])
		d.pending = d.pending[index+1:]
	}
	// Keep incomplete command lines out of the prompt too. A newline or flush
	// commits the remainder without redrawing it into scrollback.
	d.terminal.Refresh()
}

func (d *chatDisplay) flush() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.terminal.SetPrompt(d.prompt())
	if d.markdown.Len() > 0 {
		width := d.width()
		fmt.Fprintln(d.terminal.Stdout(), renderMarkdown(d.markdown.String(), width))
		d.markdown.Reset()
	}
	if d.pending != "" {
		fmt.Fprintln(d.terminal.Stdout(), d.pending)
		d.pending = ""
	}
	d.terminal.Refresh()
}

func (d *chatDisplay) prompt() string {
	name := d.name
	if name == "" {
		name = "micro"
	}
	if d.busy {
		return name + " (working) > "
	}
	return name + " > "
}

func (d *chatDisplay) setBusy(busy bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.busy = busy
	d.terminal.SetPrompt(d.prompt())
	d.terminal.Refresh()
}

func (d *chatDisplay) setAgent(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.name = terminalText(name)
	d.terminal.SetPrompt(d.prompt())
	d.terminal.Refresh()
}

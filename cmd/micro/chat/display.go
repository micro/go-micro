package chat

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/chzyer/readline"
)

// chatDisplay previews replies above the editable prompt, then commits the
// formatted reply to scrollback. Tool output stays plain text.
type chatDisplay struct {
	terminal *readline.Instance
	mu       sync.Mutex
	pending  string
	markdown strings.Builder
	updated  time.Time
	busy     bool
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
	// Avoid re-rendering the reply for every token in a fast provider stream.
	if time.Since(d.updated) < 80*time.Millisecond {
		return
	}
	d.updated = time.Now()
	width, height := d.size()
	rendered := renderMarkdown(d.markdown.String(), width)
	rows := min(6, max(1, height/3))
	preview := markdownPreview(rendered, width, rows)
	d.terminal.SetPrompt(preview + "\n" + d.prompt())
	d.terminal.Refresh()
}

func (d *chatDisplay) size() (int, int) {
	width, height, err := readline.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 10 || height < 3 {
		return 80, 24
	}
	return width - 1, height
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
	width, height := d.size()
	prompt := d.prompt()
	if d.pending != "" {
		prompt = markdownPreview(d.pending, width, max(1, height/3)) + "\n" + prompt
	}
	d.terminal.SetPrompt(prompt)
	d.terminal.Refresh()
}

func (d *chatDisplay) flush() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.terminal.SetPrompt(d.prompt())
	if d.markdown.Len() > 0 {
		width, _ := d.size()
		fmt.Fprintln(d.terminal.Stdout(), renderMarkdown(d.markdown.String(), width))
		d.markdown.Reset()
		d.updated = time.Time{}
	}
	if d.pending != "" {
		fmt.Fprintln(d.terminal.Stdout(), d.pending)
		d.pending = ""
	}
	d.terminal.Refresh()
}

func (d *chatDisplay) prompt() string {
	if d.busy {
		return "micro (working) > "
	}
	return "micro > "
}

func (d *chatDisplay) setBusy(busy bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.busy = busy
	d.terminal.SetPrompt(d.prompt())
	d.terminal.Refresh()
}

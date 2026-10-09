package chat

import (
	"fmt"
	"strings"
	"sync"
	"unicode"

	"github.com/chzyer/readline"
)

// chatDisplay commits complete lines and keeps the unfinished line above the
// editable prompt. Incoming tokens never overwrite what the user is typing.
type chatDisplay struct {
	terminal *readline.Instance
	mu       sync.Mutex
	pending  string
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
	d.pending += terminalText(text)
	if index := strings.LastIndexByte(d.pending, '\n'); index >= 0 {
		d.terminal.SetPrompt("micro > ")
		fmt.Fprint(d.terminal.Stdout(), d.pending[:index+1])
		d.pending = d.pending[index+1:]
	}
	prompt := "micro > "
	if d.pending != "" {
		prompt = d.pending + "\n" + prompt
	}
	d.terminal.SetPrompt(prompt)
	d.terminal.Refresh()
}

func (d *chatDisplay) flush() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.terminal.SetPrompt("micro > ")
	if d.pending != "" {
		fmt.Fprintln(d.terminal.Stdout(), d.pending)
		d.pending = ""
	}
	d.terminal.Refresh()
}

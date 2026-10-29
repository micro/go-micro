package chat

import (
	"os"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	terminalansi "github.com/charmbracelet/x/ansi"
)

// Use the terminal's own foreground and background, so light and dark terminals
// remain readable without probing the terminal or selecting a theme.
func renderMarkdown(text string, width int) string {
	// Bound parsing work for unusually large replies; retain the full text.
	if len(text) > 64*1024 {
		return text
	}
	style := styles.ASCIIStyleConfig
	zero := uint(0)
	yes := true
	style.Document.Margin = &zero
	style.Document.BlockPrefix = ""
	style.Document.BlockSuffix = ""
	style.Heading.Bold = &yes
	style.H1, style.H2, style.H3 = ansi.StyleBlock{}, ansi.StyleBlock{}, ansi.StyleBlock{}
	style.H4, style.H5, style.H6 = ansi.StyleBlock{}, ansi.StyleBlock{}, ansi.StyleBlock{}
	style.Strong = ansi.StylePrimitive{Bold: &yes}
	style.Emph = ansi.StylePrimitive{Italic: &yes}
	style.Strikethrough = ansi.StylePrimitive{CrossedOut: &yes}
	style.LinkText = ansi.StylePrimitive{Underline: &yes}
	renderer, err := glamour.NewTermRenderer(glamour.WithStyles(style), glamour.WithWordWrap(width))
	if err != nil {
		return text
	}
	rendered, err := renderer.Render(text)
	if err != nil {
		return text
	}
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		rendered = terminalansi.Strip(rendered)
	}
	return strings.Trim(rendered, "\n")
}

// completeMarkdown returns the end of the last complete block. Keep fenced code
// together so a blank line inside a code block cannot trigger an early render.
func completeMarkdown(text string) int {
	end, offset := 0, 0
	var fence byte
	length := 0
	for {
		newline := strings.IndexByte(text[offset:], '\n')
		if newline < 0 {
			return end
		}
		next := offset + newline + 1
		line := strings.TrimSpace(text[offset:next])
		if len(line) >= 3 && (line[0] == '`' || line[0] == '~') {
			count := 0
			for count < len(line) && line[count] == line[0] {
				count++
			}
			if count >= 3 {
				if fence == 0 {
					fence, length = line[0], count
				} else if line[0] == fence && count >= length && strings.TrimSpace(line[count:]) == "" {
					fence, length = 0, 0
					end = next
				}
			}
		} else if line == "" && fence == 0 {
			end = next
		}
		offset = next
	}
}

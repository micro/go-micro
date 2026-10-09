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

// Keep the live preview small enough for the editable prompt to stay on screen.
// The complete reply is committed to scrollback when the turn finishes.
func markdownPreview(rendered string, width, rows int) string {
	lines := strings.Split(rendered, "\n")
	if len(lines) > rows {
		lines = lines[len(lines)-rows:]
	}
	for i, line := range lines {
		lines[i] = terminalansi.Truncate(line, width, "…")
	}
	return strings.Join(lines, "\n")
}

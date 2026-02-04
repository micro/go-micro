package logger

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"

	dlog "go-micro.dev/v5/debug/log"
)

// debugLogHandler is a slog handler that writes to the debug/log buffer
type debugLogHandler struct {
	level slog.Leveler
	attrs []slog.Attr
	group string
}

// newDebugLogHandler creates a new handler that writes to debug/log
func newDebugLogHandler(level slog.Leveler) *debugLogHandler {
	return &debugLogHandler{
		level: level,
		attrs: make([]slog.Attr, 0),
	}
}

func (h *debugLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *debugLogHandler) Handle(_ context.Context, r slog.Record) error {
	// Build metadata from attributes
	metadata := make(map[string]string)

	// Add handler's attributes
	for _, attr := range h.attrs {
		metadata[attr.Key] = attr.Value.String()
	}

	// Add record's attributes
	r.Attrs(func(a slog.Attr) bool {
		metadata[a.Key] = a.Value.String()
		return true
	})

	// Add level to metadata
	metadata["level"] = r.Level.String()

	// Add source if available
	if sourcePath := extractSourceFilePath(r.PC); sourcePath != "" {
		metadata["file"] = sourcePath
	}

	// Create debug log record
	rec := dlog.Record{
		Timestamp: r.Time,
		Message:   r.Message,
		Metadata:  metadata,
	}

	// Write to debug log
	_ = dlog.DefaultLog.Write(rec)

	return nil
}

func (h *debugLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &debugLogHandler{
		level: h.level,
		attrs: newAttrs,
		group: h.group,
	}
}

func (h *debugLogHandler) WithGroup(name string) slog.Handler {
	// For simplicity, we'll just track the group name
	// A full implementation would nest attributes properly
	return &debugLogHandler{
		level: h.level,
		attrs: h.attrs,
		group: name,
	}
}

// multiHandler sends records to multiple handlers
type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) *multiHandler {
	return &multiHandler{
		handlers: handlers,
	}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Enabled if any handler is enabled
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		// Clone the record for each handler to avoid issues
		if err := handler.Handle(ctx, r.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: newHandlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: newHandlers}
}

// extractSourceFilePath extracts the package/file:line from a PC
func extractSourceFilePath(pc uintptr) string {
	if pc == 0 {
		return ""
	}

	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	if f.File == "" {
		return ""
	}

	// Extract just filename, not full path
	idx := strings.LastIndexByte(f.File, '/')
	if idx == -1 {
		return fmt.Sprintf("%s:%d", f.File, f.Line)
	}

	// Get package/file:line
	idx2 := strings.LastIndexByte(f.File[:idx], '/')
	if idx2 == -1 {
		return fmt.Sprintf("%s:%d", f.File[idx+1:], f.Line)
	}

	return fmt.Sprintf("%s:%d", f.File[idx2+1:], f.Line)
}

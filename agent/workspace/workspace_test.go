package workspace

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestFilesStayInWorkspace(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
		t.Skip(err)
	}
	for _, path := range []string{outside, "../secret", "link"} {
		if _, err := w.read(context.Background(), map[string]any{"path": path}); err == nil {
			t.Fatalf("read escaped: %s", path)
		}
		if _, err := w.write(context.Background(), map[string]any{"path": path, "content": "changed"}); err == nil {
			t.Fatalf("write escaped: %s", path)
		}
	}
	data, err := os.ReadFile(outside)
	if err != nil || string(data) != "private" {
		t.Fatalf("outside changed: %s %v", data, err)
	}
}

func TestEditAndSearch(t *testing.T) {
	w, _ := New(t.TempDir())
	ctx := context.Background()
	if _, err := w.write(ctx, map[string]any{"path": "main.go", "content": "old old"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.edit(ctx, map[string]any{"path": "main.go", "old_text": "old", "new_text": "new"}); err == nil {
		t.Fatal("ambiguous edit accepted")
	}
	if _, err := w.edit(ctx, map[string]any{"path": "main.go", "old_text": "old old", "new_text": "new"}); err != nil {
		t.Fatal(err)
	}
	result, err := w.search(ctx, map[string]any{"query": "new"})
	if err != nil || !strings.Contains(result, "main.go:1:new") {
		t.Fatalf("search: %q %v", result, err)
	}
	instructions, err := w.Instructions()
	if err != nil || instructions != "" {
		t.Fatalf("absent instructions: %q %v", instructions, err)
	}
	if _, err := w.write(ctx, map[string]any{"path": "AGENTS.md", "content": "Run the tests."}); err != nil {
		t.Fatal(err)
	}
	instructions, err = w.Instructions()
	if err != nil || instructions != "Run the tests." {
		t.Fatalf("instructions: %q %v", instructions, err)
	}
}

func TestCommandCancellationAndOutputLimit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix shell fixture")
	}
	w, _ := New(t.TempDir())
	result, err := w.run(context.Background(), map[string]any{"command": "printf success"})
	if err != nil || result != "success" {
		t.Fatalf("command: %q %v", result, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = w.run(ctx, map[string]any{"command": "sleep 30 & wait"})
	if err == nil || time.Since(start) > 3*time.Second {
		t.Fatalf("cancellation: %v in %s", err, time.Since(start))
	}
	result, err = w.run(context.Background(), map[string]any{"command": "head -c 100000 /dev/zero"})
	if err != nil || len(result) > maxOutput+100 || !strings.Contains(result, "[Output truncated]") {
		t.Fatalf("output limit: bytes=%d err=%v", len(result), err)
	}
}

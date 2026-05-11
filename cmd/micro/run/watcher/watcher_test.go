package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDefaults(t *testing.T) {
	w := New([]string{"."})
	if w.interval != 500*time.Millisecond {
		t.Errorf("interval = %v, want 500ms", w.interval)
	}
	if w.debounce != 300*time.Millisecond {
		t.Errorf("debounce = %v, want 300ms", w.debounce)
	}
	if len(w.extensions) != 1 || w.extensions[0] != ".go" {
		t.Errorf("extensions = %v, want [\".go\"]", w.extensions)
	}
}

func TestWithOptions(t *testing.T) {
	w := New([]string{"."},
		WithInterval(1*time.Second),
		WithDebounce(500*time.Millisecond),
		WithExtensions(".go", ".mod", ".proto"),
		WithExcludes("dist", "build"),
	)
	if w.interval != 1*time.Second {
		t.Errorf("interval = %v, want 1s", w.interval)
	}
	if w.debounce != 500*time.Millisecond {
		t.Errorf("debounce = %v, want 500ms", w.debounce)
	}
	if len(w.extensions) != 3 {
		t.Errorf("extensions count = %d, want 3", len(w.extensions))
	}
	// default excludes + custom
	foundDist := false
	for _, e := range w.excludes {
		if e == "dist" {
			foundDist = true
		}
	}
	if !foundDist {
		t.Error("excludes should contain 'dist'")
	}
}

func TestMatchesExtension(t *testing.T) {
	w := New([]string{"."}, WithExtensions(".go", ".mod", ".proto"))

	tests := []struct {
		path  string
		match bool
	}{
		{"main.go", true},
		{"handler_test.go", true},
		{"go.mod", true},
		{"go.sum", true},
		{"service.proto", true},
		{"README.md", false},
		{"style.css", false},
		{"data.json", false},
	}

	for _, tt := range tests {
		if got := w.matchesExtension(tt.path); got != tt.match {
			t.Errorf("matchesExtension(%q) = %v, want %v", tt.path, got, tt.match)
		}
	}
}

func TestScanDetectsChanges(t *testing.T) {
	dir := t.TempDir()

	// Create a .go file
	goFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	w := New([]string{dir})

	// Initial scan
	w.scan(false)

	// No changes yet
	changed := w.scan(true)
	if len(changed) != 0 {
		t.Errorf("expected no changes, got %v", changed)
	}

	// Touch the file
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(goFile, []byte("package main\n// changed"), 0644); err != nil {
		t.Fatal(err)
	}

	changed = w.scan(true)
	if len(changed) == 0 {
		t.Error("expected changes after modifying file")
	}
}

func TestScanSkipsExcluded(t *testing.T) {
	dir := t.TempDir()

	// Create vendor dir with a .go file
	vendorDir := filepath.Join(dir, "vendor")
	os.MkdirAll(vendorDir, 0755)
	os.WriteFile(filepath.Join(vendorDir, "lib.go"), []byte("package lib"), 0644)

	// Create a regular .go file
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

	w := New([]string{dir})
	w.scan(false)

	// Verify vendor file is not tracked
	w.mu.Lock()
	for path := range w.modTimes {
		if filepath.Base(filepath.Dir(path)) == "vendor" {
			t.Errorf("vendor file should be excluded: %s", path)
		}
	}
	w.mu.Unlock()
}

func TestStartStop(t *testing.T) {
	w := New([]string{t.TempDir()}, WithInterval(50*time.Millisecond))
	w.Start()
	time.Sleep(100 * time.Millisecond)
	w.Stop()
}

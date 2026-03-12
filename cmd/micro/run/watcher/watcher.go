// Package watcher provides file watching for hot reload
package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Default file extensions to watch
var defaultExtensions = []string{".go"}

// Default directories to skip
var defaultExcludes = []string{"vendor", "node_modules", "testdata"}

// Event represents a file change event
type Event struct {
	Path string
	Dir  string // The service directory that was affected
}

// Watcher watches directories for file changes
type Watcher struct {
	dirs       []string
	events     chan Event
	done       chan struct{}
	interval   time.Duration
	debounce   time.Duration
	extensions []string
	excludes   []string

	mu       sync.Mutex
	modTimes map[string]time.Time
}

// Option configures the watcher
type Option func(*Watcher)

// WithInterval sets the polling interval
func WithInterval(d time.Duration) Option {
	return func(w *Watcher) {
		w.interval = d
	}
}

// WithDebounce sets the debounce duration for rapid changes
func WithDebounce(d time.Duration) Option {
	return func(w *Watcher) {
		w.debounce = d
	}
}

// WithExtensions sets the file extensions to watch (e.g., ".go", ".mod", ".proto").
// Replaces the default list. Each extension must include the leading dot.
func WithExtensions(exts ...string) Option {
	return func(w *Watcher) {
		w.extensions = exts
	}
}

// WithExcludes sets additional directory names to skip during scanning.
// These are added to the default excludes (vendor, node_modules, testdata).
func WithExcludes(dirs ...string) Option {
	return func(w *Watcher) {
		w.excludes = append(w.excludes, dirs...)
	}
}

// New creates a new file watcher for the given directories
func New(dirs []string, opts ...Option) *Watcher {
	w := &Watcher{
		dirs:       dirs,
		events:     make(chan Event, 100),
		done:       make(chan struct{}),
		interval:   500 * time.Millisecond,
		debounce:   300 * time.Millisecond,
		extensions: append([]string{}, defaultExtensions...),
		excludes:   append([]string{}, defaultExcludes...),
		modTimes:   make(map[string]time.Time),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// Events returns the channel of file change events
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Start begins watching for file changes
func (w *Watcher) Start() {
	// Initial scan to populate mod times
	w.scan(false)

	go w.watch()
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.done)
}

func (w *Watcher) watch() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Track pending events per directory for debouncing
	pending := make(map[string]time.Time)
	var pendingMu sync.Mutex

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			changed := w.scan(true)
			now := time.Now()

			pendingMu.Lock()
			for _, dir := range changed {
				pending[dir] = now
			}

			// Emit events for directories that have been stable
			for dir, t := range pending {
				if now.Sub(t) >= w.debounce {
					select {
					case w.events <- Event{Dir: dir}:
					default:
						// Channel full, skip
					}
					delete(pending, dir)
				}
			}
			pendingMu.Unlock()
		}
	}
}

func (w *Watcher) scan(notify bool) []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	var changed []string
	changedDirs := make(map[string]bool)

	// Build exclude set for O(1) lookup
	excludeSet := make(map[string]bool, len(w.excludes))
	for _, e := range w.excludes {
		excludeSet[e] = true
	}

	for _, dir := range w.dirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}

		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			// Skip hidden directories and excluded dirs
			if info.IsDir() {
				name := info.Name()
				if strings.HasPrefix(name, ".") || excludeSet[name] {
					return filepath.SkipDir
				}
				return nil
			}

			// Check if file matches any watched extension
			if !w.matchesExtension(path) {
				return nil
			}

			modTime := info.ModTime()
			if oldTime, exists := w.modTimes[path]; exists {
				if modTime.After(oldTime) && notify {
					if !changedDirs[absDir] {
						changedDirs[absDir] = true
						changed = append(changed, absDir)
					}
				}
			}
			w.modTimes[path] = modTime

			return nil
		})
	}

	return changed
}

// matchesExtension checks if a file path has one of the watched extensions.
// Special case: "go.mod" and "go.sum" are always matched when ".mod" is in the extensions list.
func (w *Watcher) matchesExtension(path string) bool {
	base := filepath.Base(path)
	for _, ext := range w.extensions {
		if strings.HasSuffix(base, ext) {
			return true
		}
		// Special case: watch go.mod and go.sum when .mod extension is listed
		if ext == ".mod" && (base == "go.mod" || base == "go.sum") {
			return true
		}
	}
	return false
}

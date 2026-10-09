// Package workspace provides file and command tools for development agents.
// File tools stay within the root directory. Shell commands run with the host's
// permissions: this package is not a sandbox. Hosts should install an approval
// policy with agent.WithApproval before exposing these tools to a model.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"go-micro.dev/v6/agent"
)

const maxOutput = 64 * 1024

// Workspace is a directory an agent can work in.
type Workspace struct {
	dir string
	mu  sync.Mutex
}

func New(dir string) (*Workspace, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	root.Close()
	return &Workspace{dir: dir}, nil
}

// Tools registers ordinary agent tools; callers retain control over approvals,
// models, memory, and the agent lifecycle.
func (w *Workspace) Tools() []agent.Option {
	text := func(description string) any { return map[string]any{"type": "string", "description": description} }
	return []agent.Option{
		agent.WithTool("workspace_read", "Read a UTF-8 text file relative to the workspace (maximum 64 KiB).", map[string]any{"path": text("Relative file path")}, w.read),
		agent.WithTool("workspace_search", "Find literal text in workspace files. Empty query lists files. Skips hidden directories, vendor and node_modules; returns up to 100 matches.", map[string]any{"query": text("Literal text to find")}, w.search),
		agent.WithTool("workspace_write", "Create or replace a text file in the workspace. Read existing files first. Requires an existing parent directory.", map[string]any{"path": text("Relative file path"), "content": text("Complete new file content")}, w.write),
		agent.WithTool("workspace_edit", "Replace one exact, unique occurrence of old_text in a workspace file. Fails if missing or ambiguous.", map[string]any{"path": text("Relative file path"), "old_text": text("Exact text to replace"), "new_text": text("Replacement text")}, w.edit),
		agent.WithTool("workspace_exec", "Run a shell command in the workspace with host permissions. Not sandboxed. Output is limited to 64 KiB and execution to two minutes.", map[string]any{"command": text("Shell command")}, w.run),
	}
}

// Instructions reads the workspace's root AGENTS.md, if present.
func (w *Workspace) Instructions() (string, error) {
	text, err := w.read(context.Background(), map[string]any{"path": "AGENTS.md"})
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return text, err
}

func argument(input map[string]any, key string) (string, error) {
	value, ok := input[key].(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return value, nil
}

func readText(root *os.Root, path string) (string, error) {
	info, err := root.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file: %s", path)
	}
	f, err := root.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file: %s", path)
	}
	data, err := io.ReadAll(io.LimitReader(f, maxOutput+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxOutput {
		return "", fmt.Errorf("file exceeds 64 KiB: %s", path)
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return "", fmt.Errorf("binary file: %s", path)
	}
	return string(data), nil
}

func (w *Workspace) read(ctx context.Context, input map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := argument(input, "path")
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(w.dir)
	if err != nil {
		return "", err
	}
	defer root.Close()
	return readText(root, path)
}

func (w *Workspace) write(ctx context.Context, input map[string]any) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writeFile(ctx, input)
}

func (w *Workspace) writeFile(ctx context.Context, input map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := argument(input, "path")
	if err != nil {
		return "", err
	}
	content, err := argument(input, "content")
	if err != nil {
		return "", err
	}
	if len(content) > maxOutput {
		return "", errors.New("content exceeds 64 KiB")
	}
	root, err := os.OpenRoot(w.dir)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if info, err := root.Stat(path); err == nil && !info.Mode().IsRegular() {
		return "", errors.New("target is not a regular file")
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	f, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	_, err = io.WriteString(f, content)
	closeErr := f.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	return "Wrote " + path, nil
}

func (w *Workspace) edit(ctx context.Context, input map[string]any) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	old, err := argument(input, "old_text")
	if err != nil {
		return "", err
	}
	replacement, err := argument(input, "new_text")
	if err != nil {
		return "", err
	}
	if old == "" {
		return "", errors.New("old_text must not be empty")
	}
	content, err := w.read(ctx, input)
	if err != nil {
		return "", err
	}
	if strings.Count(content, old) != 1 {
		return "", errors.New("old_text must match exactly once; read the file again")
	}
	return w.writeFile(ctx, map[string]any{"path": input["path"], "content": strings.Replace(content, old, replacement, 1)})
}

func (w *Workspace) search(ctx context.Context, input map[string]any) (string, error) {
	query, err := argument(input, "query")
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(w.dir)
	if err != nil {
		return "", err
	}
	defer root.Close()
	var out strings.Builder
	matches := 0
	err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules" || d.Name() == "vendor") {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if query == "" {
			fmt.Fprintln(&out, path)
			matches++
		} else {
			content, err := readText(root, path)
			if err != nil {
				return nil
			}
			for i, line := range strings.Split(content, "\n") {
				if strings.Contains(line, query) {
					fmt.Fprintf(&out, "%s:%d:%s\n", path, i+1, line)
					matches++
				}
				if matches >= 100 || out.Len() >= maxOutput {
					break
				}
			}
		}
		if matches >= 100 || out.Len() >= maxOutput {
			return fs.SkipAll
		}
		return nil
	})
	result := out.String()
	if len(result) > maxOutput {
		result = result[:maxOutput]
	}
	if matches >= 100 || out.Len() >= maxOutput {
		result += "\n[Results truncated]"
	}
	return result, err
}

type outputBuffer struct {
	mu        sync.Mutex
	text      strings.Builder
	truncated bool
}

func (b *outputBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	remaining := maxOutput - b.text.Len()
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	b.text.Write(p)
	return n, nil
}

func (w *Workspace) run(ctx context.Context, input map[string]any) (string, error) {
	command, err := argument(input, "command")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(command) == "" {
		return "", errors.New("command must not be empty")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := shellCommand(ctx, command)
	cmd.Dir = w.dir
	cmd.WaitDelay = time.Second
	var output outputBuffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err = cmd.Start(); err != nil {
		return "", err
	}
	// Also stop background children when the shell itself exits.
	defer func() { _ = cmd.Cancel() }()
	err = cmd.Wait()
	result := output.text.String()
	if output.truncated {
		result += "\n[Output truncated]"
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if err != nil {
		return result, fmt.Errorf("command failed: %w; output: %s", err, result)
	}
	return result, nil
}

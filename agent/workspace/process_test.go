package workspace

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBackgroundProcessOwnedByWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	w, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	ctx, cancel := context.WithCancel(context.Background())
	process, err := w.Start(ctx, "printf ready; sleep 30")
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	deadline := time.Now().Add(2 * time.Second)
	for {
		processes := w.Processes()
		if len(processes) != 1 || !processes[0].Running {
			t.Fatalf("process stopped with caller: %v", processes)
		}
		if strings.Contains(processes[0].Output, "ready") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("command produced no output")
		}
		time.Sleep(time.Millisecond)
	}
	if err := w.Stop(process.ID); err != nil {
		t.Fatal(err)
	}
	if w.Processes()[0].Running {
		t.Fatal("process still running")
	}
	if _, err := w.processTool(context.Background(), map[string]any{"action": "forget", "id": process.ID}); err != nil {
		t.Fatal(err)
	}
	if len(w.Processes()) != 0 {
		t.Fatal("process not forgotten")
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Start(context.Background(), "echo no"); err == nil {
		t.Fatal("started after close")
	}
}

func TestContainerCommandRestrictsHostAccess(t *testing.T) {
	w, err := New(t.TempDir(), WithContainer("test-image"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	command := w.command(ctx, "echo hello")
	args := strings.Join(command.Args, " ")
	for _, want := range []string{"--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "dst=/workspace", "-- test-image sh -lc echo hello"} {
		if !strings.Contains(args, want) {
			t.Errorf("missing %s: %s", want, args)
		}
	}
}

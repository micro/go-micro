package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Process is a bounded snapshot of a command owned by the workspace host.
type Process struct {
	ID      string `json:"id"`
	Command string `json:"command"`
	Running bool   `json:"running"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}
type process struct {
	command *exec.Cmd
	cancel  context.CancelFunc
	done    chan struct{}
	output  outputBuffer
	state   Process
}

// Start runs a command until it exits, Stop is called, or the workspace closes.
// Unlike a foreground tool call, canceling the submitting request does not stop
// it. Hosts must call Close; background commands never outlive their host.
func (w *Workspace) Start(ctx context.Context, command string) (Process, error) {
	if err := ctx.Err(); err != nil {
		return Process{}, err
	}
	if command == "" {
		return Process{}, errors.New("command must not be empty")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return Process{}, errors.New("workspace is closed")
	}
	if len(w.processes) >= 64 {
		return Process{}, errors.New("workspace retains at most 64 processes; forget completed processes first")
	}
	if w.processes == nil {
		w.processes = map[string]*process{}
	}
	workCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	p := &process{cancel: cancel, done: make(chan struct{}), state: Process{ID: uuid.NewString(), Command: command, Running: true}}
	p.command = w.command(workCtx, command)
	p.command.Dir = w.dir
	p.command.WaitDelay = time.Second
	p.command.Stdout, p.command.Stderr = &p.output, &p.output
	if err := p.command.Start(); err != nil {
		cancel()
		return Process{}, err
	}
	w.processes[p.state.ID] = p
	go func() {
		err := p.command.Wait()
		_ = p.command.Cancel()
		cancel()
		w.mu.Lock()
		defer w.mu.Unlock()
		p.state.Running = false
		if err != nil {
			p.state.Error = err.Error()
		}
		close(p.done)
	}()
	return p.state, nil
}

// Processes reports retained commands in stable ID order.
func (w *Workspace) Processes() []Process {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make([]Process, 0, len(w.processes))
	for _, p := range w.processes {
		state := p.state
		p.output.mu.Lock()
		state.Output = p.output.text.String()
		if p.output.truncated {
			state.Output += "\n[Output truncated]"
		}
		p.output.mu.Unlock()
		result = append(result, state)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (w *Workspace) Stop(id string) error {
	w.mu.Lock()
	p := w.processes[id]
	w.mu.Unlock()
	if p == nil {
		return fmt.Errorf("unknown process %s", id)
	}
	p.cancel()
	<-p.done
	return nil
}

// Close stops all managed background commands and waits for their process trees.
func (w *Workspace) Close() error {
	w.mu.Lock()
	w.closed = true
	var running []*process
	for _, p := range w.processes {
		p.cancel()
		running = append(running, p)
	}
	w.mu.Unlock()
	for _, p := range running {
		<-p.done
	}
	return nil
}

func (w *Workspace) processTool(ctx context.Context, input map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	action, err := argument(input, "action")
	if err != nil {
		return "", err
	}
	id, _ := input["id"].(string)
	switch action {
	case "list":
	case "stop":
		if err := w.Stop(id); err != nil {
			return "", err
		}
	case "forget":
		w.mu.Lock()
		p := w.processes[id]
		if p == nil || p.state.Running {
			w.mu.Unlock()
			return "", errors.New("only a completed process can be forgotten")
		}
		delete(w.processes, id)
		w.mu.Unlock()
	default:
		return "", errors.New("action must be list, stop or forget")
	}
	data, err := json.Marshal(w.Processes())
	return string(data), err
}

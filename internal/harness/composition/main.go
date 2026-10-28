// Command composition proves the restartable service-agent-flow contract with
// real in-process RPC, a mock model, and a persistent local checkpoint.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
	"go-micro.dev/v6/service"
	"go-micro.dev/v6/store"
)

type Request struct {
	Key string `json:"key"`
}
type Response struct {
	Key  string `json:"key"`
	Done bool   `json:"done"`
}

type Capability struct {
	mu                          sync.Mutex
	prepared, worked, finalized map[string]int
}

func newCapability() *Capability {
	return &Capability{prepared: map[string]int{}, worked: map[string]int{}, finalized: map[string]int{}}
}
func (s *Capability) Prepare(_ context.Context, req *Request, rsp *Response) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prepared[req.Key]++
	rsp.Key = req.Key
	return nil
}
func (s *Capability) Work(_ context.Context, req *Request, rsp *Response) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.worked[req.Key]++
	rsp.Key, rsp.Done = req.Key, true
	return nil
}
func (s *Capability) Finalize(_ context.Context, req *Request, rsp *Response) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finalized[req.Key] == 0 {
		s.finalized[req.Key] = 1
	}
	rsp.Key, rsp.Done = req.Key, true
	return nil
}
func (s *Capability) counts(key string) (int, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.prepared[key], s.worked[key], s.finalized[key]
}

type mock struct{ opts model.Options }

func (m *mock) Init(opts ...model.Option) error {
	for _, opt := range opts {
		opt(&m.opts)
	}
	return nil
}
func (m *mock) Options() model.Options { return m.opts }
func (m *mock) String() string         { return "composition-mock" }
func (m *mock) Stream(context.Context, *model.Request, ...model.GenerateOption) (model.Stream, error) {
	return nil, errors.New("stream unsupported")
}
func (m *mock) Generate(ctx context.Context, req *model.Request, _ ...model.GenerateOption) (*model.Response, error) {
	name := ""
	for _, tool := range req.Tools {
		if strings.Contains(tool.Name, "Work") {
			name = tool.Name
			break
		}
	}
	if name == "" || m.opts.ToolHandler == nil {
		return nil, errors.New("work service tool not discovered")
	}
	key := "composition-42"
	result := m.opts.ToolHandler(ctx, model.ToolCall{Name: name, Input: map[string]any{"key": key}})
	if result.Refused != "" {
		return nil, fmt.Errorf("work tool refused: %s", result.Refused)
	}
	return &model.Response{Answer: `{"key":"composition-42","done":true}`}, nil
}

func main() {
	if err := run(context.Background()); err != nil {
		panic(err)
	}
	fmt.Println("composition fixture: PASS")
}

func run(ctx context.Context) error {
	dir, err := os.MkdirTemp("", "go-micro-composition-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	st := store.NewStore(store.DirOption(filepath.Join(dir, "state")))
	cp := flow.StoreCheckpoint(st, "composition")
	agentCP := flow.StoreCheckpoint(st, "composition-agent")

	model.Register("composition-mock", func(opts ...model.Option) model.Model { m := new(mock); _ = m.Init(opts...); return m })
	reg := registry.NewMemoryRegistry()
	cl := client.NewClient(client.Registry(reg), client.Selector(selector.NewSelector(selector.Registry(reg))))
	capability := newCapability()
	svc := service.New(service.Name("composition-capability"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	if err := svc.Handle(capability); err != nil {
		return err
	}
	go svc.Run()
	defer svc.Stop()

	ag := agent.New(agent.Name("composition-agent"), agent.Address("127.0.0.1:0"), agent.Provider("composition-mock"), agent.Services("composition-capability"), agent.MaxSteps(1), agent.WithRegistry(reg), agent.WithClient(cl), agent.WithStore(st), agent.WithCheckpoint(agentCP))
	go ag.Run()
	defer ag.Stop()
	if err := waitFor(ctx, reg, "composition-capability", "composition-agent"); err != nil {
		return err
	}

	crash := map[string]bool{"before-agent": true, "before-approval": true, "finalize": true}
	before := func(name string, fn flow.StepFunc) flow.StepFunc {
		return func(ctx context.Context, in flow.State) (flow.State, error) {
			if crash[name] {
				crash[name] = false
				return in, fmt.Errorf("injected host stop at %s", name)
			}
			return fn(ctx, in)
		}
	}
	after := func(name string, fn flow.StepFunc) flow.StepFunc {
		return func(ctx context.Context, in flow.State) (flow.State, error) {
			out, err := fn(ctx, in)
			if err == nil && crash[name] {
				crash[name] = false
				return out, fmt.Errorf("injected host stop after %s", name)
			}
			return out, err
		}
	}
	steps := []flow.Step{
		{Name: "prepare", Run: flow.Call("composition-capability", "Capability.Prepare")},
		{Name: "agent", Run: before("before-agent", flow.Dispatch("composition-agent"))},
		{Name: "approval", Run: before("before-approval", flow.Await("approval", "approve final operation"))},
		{Name: "finalize", Run: after("finalize", flow.Call("composition-capability", "Capability.Finalize"))},
	}
	newHost := func() *flow.Flow {
		f := flow.New("composition", flow.Agent("composition-agent"), flow.Steps(steps...), flow.WithCheckpoint(cp))
		_ = f.Register(reg, broker.NewMemoryBroker(), cl)
		return f
	}
	root := model.WithRunInfo(ctx, model.RunInfo{RunID: "request-composition", Dispatch: "direct"})
	if err := newHost().Execute(root, `{"key":"composition-42"}`); err == nil {
		return errors.New("prepare boundary did not stop")
	}
	runs, _ := cp.List(ctx)
	if len(runs) != 1 {
		return fmt.Errorf("runs=%d, want 1", len(runs))
	}
	id := runs[0].ID
	if err := newHost().Resume(ctx, id); err == nil {
		return errors.New("agent boundary did not stop")
	}
	if err := newHost().Resume(ctx, id); err != nil {
		return fmt.Errorf("resume to approval: %w", err)
	}
	waiting, _ := newHost().Waiting(ctx)
	if len(waiting) != 1 || waiting[0].Await == nil || waiting[0].Await.Prompt == "" {
		return errors.New("approval was not durably inspectable")
	}
	if err := newHost().ResumeWith(ctx, id, `{"key":"composition-42"}`); err == nil {
		return errors.New("final boundary did not stop")
	}
	if err := newHost().Resume(ctx, id); err != nil {
		return fmt.Errorf("final restart: %w", err)
	}
	done, ok, _ := cp.Load(ctx, id)
	if !ok || done.Status != "done" || done.ParentID != "request-composition" {
		return fmt.Errorf("lost status or lineage: %+v", done)
	}
	agentRuns, _ := agentCP.List(ctx)
	if len(agentRuns) == 0 || agentRuns[len(agentRuns)-1].ParentID != id {
		return fmt.Errorf("agent child lineage was not inspectable: %+v", agentRuns)
	}
	prepared, worked, finalized := capability.counts("composition-42")
	if prepared != 1 || worked != 1 || finalized != 1 {
		return fmt.Errorf("unexpected effects prepare=%d work=%d finalize=%d", prepared, worked, finalized)
	}

	failing := flow.New("composition-failed", flow.Steps(flow.Step{Name: "failed-tool", Run: func(_ context.Context, in flow.State) (flow.State, error) { return in, errors.New("tool failed") }}), flow.WithCheckpoint(flow.StoreCheckpoint(st, "failed")))
	if err := failing.Execute(ctx, "x"); err == nil {
		return errors.New("failed tool reported success")
	}
	cancelCP := flow.StoreCheckpoint(st, "cancelled")
	cancelled := flow.New("composition-cancelled", flow.Steps(flow.Step{Name: "cancelled", Run: func(_ context.Context, in flow.State) (flow.State, error) { return in, context.Canceled }}), flow.WithCheckpoint(cancelCP))
	if err := cancelled.Execute(ctx, "x"); !errors.Is(err, context.Canceled) {
		return fmt.Errorf("cancel outcome=%v", err)
	}
	cancelledRuns, _ := cancelCP.List(ctx)
	if len(cancelledRuns) != 1 || cancelledRuns[0].Status != "failed" || cancelledRuns[0].Steps[0].ErrorKind != string(model.ErrorKindCanceled) {
		return fmt.Errorf("cancelled run was not a terminal non-success: %+v", cancelledRuns)
	}
	return nil
}

func waitFor(ctx context.Context, reg registry.Registry, names ...string) error {
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		all := true
		for _, name := range names {
			svcs, err := reg.GetService(name)
			if err != nil || len(svcs) == 0 || len(svcs[0].Nodes) == 0 {
				all = false
			}
		}
		if all {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("services did not register")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

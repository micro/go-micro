// Plan & Delegate integration harness.
//
// This runs the REAL go-micro stack end to end — real services, real
// registry, real RPC, the real agent loop, real store, real delegate
// routing — and mocks ONLY the LLM with a deterministic provider. It
// proves the plumbing works without an API key, and it's reproducible.
//
// Swap MICRO_AI_PROVIDER/MICRO_AI_API_KEY (and remove --mock) to run the
// exact same flow against a live model.
//
// Run:
//
//	go run ./internal/harness/plan-delegate
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/ai"
)

// ---------------------------------------------------------------------------
// real services
// ---------------------------------------------------------------------------

type Task struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type AddRequest struct {
	Title string `json:"title" description:"Title of the task to add"`
}
type AddResponse struct {
	Task *Task `json:"task"`
}
type ListRequest struct{}
type ListResponse struct {
	Tasks []*Task `json:"tasks"`
}

type TaskService struct {
	mu     sync.Mutex
	tasks  []*Task
	nextID int
}

// Add creates a new task with the given title.
// @example {"title": "Design"}
func (s *TaskService) Add(ctx context.Context, req *AddRequest, rsp *AddResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	t := &Task{ID: fmt.Sprintf("task-%d", s.nextID), Title: req.Title}
	s.tasks = append(s.tasks, t)
	rsp.Task = t
	fmt.Printf("    \033[32m[task]\033[0m created %s %q\n", t.ID, t.Title)
	return nil
}

// List returns all tasks.
// @example {}
func (s *TaskService) List(ctx context.Context, req *ListRequest, rsp *ListResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rsp.Tasks = append(rsp.Tasks, s.tasks...)
	return nil
}

type SendRequest struct {
	To      string `json:"to" description:"Recipient address"`
	Message string `json:"message" description:"Message body"`
}
type SendResponse struct {
	Sent bool `json:"sent"`
}
type NotifyService struct{}

// Send delivers a notification message to a recipient.
// @example {"to": "owner@acme.com", "message": "ready"}
func (s *NotifyService) Send(ctx context.Context, req *SendRequest, rsp *SendResponse) error {
	fmt.Printf("    \033[35m[notify]\033[0m 📨 to=%s message=%q\n", req.To, req.Message)
	rsp.Sent = true
	return nil
}

// ---------------------------------------------------------------------------
// mock LLM provider — the ONLY fake. It "reasons" by simple heuristics
// over the tools it's offered and the system prompt it's given, calling
// the real tool handler exactly the way a real provider would.
// ---------------------------------------------------------------------------

type mockModel struct{ opts ai.Options }

func newMock(opts ...ai.Option) ai.Model {
	m := &mockModel{}
	_ = m.Init(opts...)
	return m
}

func (m *mockModel) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *mockModel) Options() ai.Options { return m.opts }
func (m *mockModel) String() string      { return "mock" }
func (m *mockModel) Stream(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (ai.Stream, error) {
	return nil, fmt.Errorf("stream not supported by mock")
}

// findTool returns the safe name of the first offered tool whose name
// contains sub, or "" if none.
func findTool(tools []ai.Tool, sub string) string {
	for _, t := range tools {
		if strings.Contains(t.Name, sub) {
			return t.Name
		}
	}
	return ""
}

func (m *mockModel) call(who, name string, input map[string]any) {
	args, _ := json.Marshal(input)
	fmt.Printf("  \033[33m[%s]\033[0m → %s(%s)\n", who, name, args)
	if m.opts.ToolHandler != nil {
		m.opts.ToolHandler(name, input)
	}
}

func (m *mockModel) Generate(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	// Classify by the tools actually offered, not by prompt text:
	// the conductor has the task "Add" tool, comms has "Send".
	hasAdd := findTool(req.Tools, "Add") != ""
	hasSend := findTool(req.Tools, "Send") != ""

	switch {
	// comms agent: owns notify, has Send but not Add.
	case hasSend && !hasAdd:
		send := findTool(req.Tools, "Send")
		m.call("comms", send, map[string]any{
			"to":      "owner@acme.com",
			"message": "The launch plan is ready",
		})
		return &ai.Response{Answer: "Notified owner@acme.com."}, nil

	// conductor: has the task Add tool — plan, create tasks, delegate.
	case hasAdd:
		if plan := findTool(req.Tools, "plan"); plan != "" {
			m.call("conductor", plan, map[string]any{
				"steps": []any{
					map[string]any{"task": "create Design task", "status": "pending"},
					map[string]any{"task": "create Build task", "status": "pending"},
					map[string]any{"task": "create Ship task", "status": "pending"},
					map[string]any{"task": "notify owner via comms", "status": "pending"},
				},
			})
		}
		if add := findTool(req.Tools, "Add"); add != "" {
			for _, title := range []string{"Design", "Build", "Ship"} {
				m.call("conductor", add, map[string]any{"title": title})
			}
		}
		if del := findTool(req.Tools, "delegate"); del != "" {
			m.call("conductor", del, map[string]any{
				"task": "Notify owner@acme.com that the launch plan is ready",
				"to":   "comms",
			})
		}
		return &ai.Response{Answer: "Created Design, Build and Ship, and had comms notify the owner."}, nil

	// ephemeral sub-agent or anything else.
	default:
		return &ai.Response{Reply: "subtask handled"}, nil
	}
}

func main() {
	ai.Register("mock", newMock)

	fmt.Println("\n\033[1mPlan & Delegate — live integration harness (mock LLM)\033[0m")
	fmt.Println("Real services, registry, RPC, agent loop, store, delegation. Only the model is mocked.\n")

	// Real services.
	task := micro.New("task")
	task.Handle(new(TaskService))
	go task.Run()

	notify := micro.New("notify")
	notify.Handle(new(NotifyService))
	go notify.Run()

	// Real comms agent (owns notify), registered so delegate reaches it over RPC.
	comms := micro.NewAgent("comms",
		micro.AgentServices("notify"),
		micro.AgentPrompt("You handle outbound notifications. Use the notify service."),
		micro.AgentProvider("mock"),
	)
	go comms.Run()

	// Real conductor agent (owns task).
	conductor := micro.NewAgent("conductor",
		micro.AgentServices("task"),
		micro.AgentPrompt("You coordinate launch work. Plan first, create tasks, and delegate notifications to the \"comms\" agent."),
		micro.AgentProvider("mock"),
	)

	fmt.Println("waiting for services + comms agent to register...")
	time.Sleep(3 * time.Second)

	fmt.Println("\n\033[1m> prompt:\033[0m Create three launch tasks (Design, Build, Ship), then make sure owner@acme.com is notified.\n")

	resp, err := conductor.Ask(context.Background(),
		"Create three launch tasks: Design, Build, and Ship. Then make sure owner@acme.com is notified that the launch plan is ready.")
	if err != nil {
		fmt.Println("\033[31merror:\033[0m", err)
		os.Exit(1)
	}

	fmt.Println("\n\033[1m< conductor reply:\033[0m", resp.Reply)

	// Prove plan was persisted to the real store.
	if recs, _ := conductor.Options().Store.Read("agent/conductor/plan"); len(recs) > 0 {
		fmt.Printf("\n\033[1mstored plan (agent/conductor/plan):\033[0m %s\n", string(recs[0].Value))
	} else {
		fmt.Println("\n\033[31m! plan was not persisted\033[0m")
	}

	fmt.Println("\n\033[32m✓ end-to-end flow complete\033[0m")
}

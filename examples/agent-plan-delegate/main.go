// Agent Plan & Delegate — planning and multi-agent delegation
//
// This example shows the two built-in agent capabilities:
//
//   - plan:     the conductor records an ordered plan before doing
//     multi-step work; the plan is saved to its memory.
//   - delegate: the conductor hands the notification step to a
//     separate "comms" agent over RPC, rather than doing it
//     itself.
//
// Two services (task, notify), two agents (conductor, comms). The
// conductor manages task; comms manages notify. When asked to create
// tasks and notify someone, the conductor plans the work, creates the
// tasks with its own tools, then delegates the notification to comms —
// which is a real registered agent, so the hand-off goes over RPC.
//
// Run (needs an LLM provider key):
//
//	MICRO_AI_PROVIDER=anthropic MICRO_AI_API_KEY=sk-ant-... go run main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go-micro.dev/v6"
)

// ---------------------------------------------------------------------------
// task service
// ---------------------------------------------------------------------------

type Task struct {
	ID    string `json:"id" description:"Unique task identifier"`
	Title string `json:"title" description:"What the task is"`
}

type AddRequest struct {
	Title string `json:"title" description:"Title of the task to add (required)"`
}

type AddResponse struct {
	Task *Task `json:"task" description:"The created task"`
}

type ListRequest struct{}

type ListResponse struct {
	Tasks []*Task `json:"tasks" description:"All tasks"`
}

type TaskService struct {
	mu     sync.Mutex
	tasks  []*Task
	nextID int
}

// Add creates a new task with the given title.
//
// @example {"title": "Design the launch page"}
func (s *TaskService) Add(ctx context.Context, req *AddRequest, rsp *AddResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	t := &Task{ID: fmt.Sprintf("task-%d", s.nextID), Title: req.Title}
	s.tasks = append(s.tasks, t)
	rsp.Task = t
	return nil
}

// List returns all tasks.
//
// @example {}
func (s *TaskService) List(ctx context.Context, req *ListRequest, rsp *ListResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rsp.Tasks = append(rsp.Tasks, s.tasks...)
	return nil
}

// ---------------------------------------------------------------------------
// notify service
// ---------------------------------------------------------------------------

type SendRequest struct {
	To      string `json:"to" description:"Recipient address (required)"`
	Message string `json:"message" description:"Message body (required)"`
}

type SendResponse struct {
	Sent bool `json:"sent" description:"Whether the notification was sent"`
}

type NotifyService struct{}

// Send delivers a notification message to a recipient.
//
// @example {"to": "owner@acme.com", "message": "The launch plan is ready"}
func (s *NotifyService) Send(ctx context.Context, req *SendRequest, rsp *SendResponse) error {
	fmt.Printf("  📨 notify: to=%s message=%q\n", req.To, req.Message)
	rsp.Sent = true
	return nil
}

func main() {
	provider, apiKey := detectProvider()
	if apiKey == "" {
		fmt.Println("No LLM key found. Set a provider key and run again, e.g.:")
		fmt.Println("  export ANTHROPIC_API_KEY=sk-ant-...   # or OPENAI_API_KEY, GEMINI_API_KEY, ...")
		fmt.Println("  go run main.go")
		return
	}
	fmt.Printf("Using provider %q\n", provider)

	// Services.
	task := micro.NewService("task")
	task.Handle(new(TaskService))
	go task.Run()

	notify := micro.NewService("notify")
	notify.Handle(new(NotifyService))
	go notify.Run()

	// comms is a real, registered agent that owns the notify service.
	// Because it's registered, the conductor's delegate hand-off
	// reaches it over RPC.
	comms := micro.NewAgent("comms",
		micro.AgentServices("notify"),
		micro.AgentPrompt("You handle outbound notifications. Use the notify service to send messages."),
		micro.AgentProvider(provider),
		micro.AgentAPIKey(apiKey),
	)
	go comms.Run()

	// The conductor owns task. Its prompt nudges it to plan, and to
	// delegate notifications to the comms agent rather than doing them.
	conductor := micro.NewAgent("conductor",
		micro.AgentServices("task"),
		micro.AgentPrompt(
			"You coordinate launch work. For multi-step requests, first call the plan tool "+
				"to record your steps, then carry them out. You can create tasks yourself. "+
				"For anything to do with notifying people, delegate to the \"comms\" agent "+
				"using the delegate tool (to: \"comms\") — do not try to notify directly.",
		),
		micro.AgentProvider(provider),
		micro.AgentAPIKey(apiKey),
	)

	// Give the services and comms agent a moment to register.
	time.Sleep(2 * time.Second)

	resp, err := conductor.Ask(context.Background(),
		"Create three launch tasks: Design, Build, and Ship. "+
			"Then make sure owner@acme.com is notified that the launch plan is ready.")
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	fmt.Println("\n--- conductor tool calls ---")
	for _, tc := range resp.ToolCalls {
		args, _ := json.Marshal(tc.Input)
		fmt.Printf("  → %s(%s)\n", tc.Name, args)
	}
	fmt.Println("\n--- conductor reply ---")
	fmt.Println(resp.Reply)
}

// detectProvider picks an LLM provider and key from the environment.
// MICRO_AI_PROVIDER / MICRO_AI_API_KEY win if set; otherwise it falls
// back to the first provider-specific key it finds (ANTHROPIC_API_KEY,
// OPENAI_API_KEY, ...), so `export ANTHROPIC_API_KEY=... && go run .`
// just works.
func detectProvider() (provider, apiKey string) {
	provider = os.Getenv("MICRO_AI_PROVIDER")
	apiKey = os.Getenv("MICRO_AI_API_KEY")
	if apiKey != "" {
		if provider == "" {
			provider = "anthropic"
		}
		return provider, apiKey
	}

	// provider name -> its conventional API key env var
	for _, p := range []struct{ name, env string }{
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"openai", "OPENAI_API_KEY"},
		{"gemini", "GEMINI_API_KEY"},
		{"groq", "GROQ_API_KEY"},
		{"mistral", "MISTRAL_API_KEY"},
		{"together", "TOGETHER_API_KEY"},
		{"atlascloud", "ATLASCLOUD_API_KEY"},
	} {
		if v := os.Getenv(p.env); v != "" {
			return p.name, v
		}
	}
	return "", ""
}

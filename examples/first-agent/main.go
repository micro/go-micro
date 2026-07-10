// First Agent — the smallest runnable service-backed agent.
//
// Run:
//
//	go run ./examples/first-agent
//
// It uses a deterministic mock model, so it needs no provider API key. The
// point is to show the first agent shape: a service exposes a tool, an agent
// discovers that service, the model asks to call the tool, and the agent returns
// a final answer.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
	"go-micro.dev/v6/service"
	"go-micro.dev/v6/store"
)

type ListNotesRequest struct{}

type ListNotesResponse struct {
	Notes []string `json:"notes" description:"Notes the assistant can summarize"`
}

type NotesService struct{ w io.Writer }

// List returns the starter notes the first agent can read.
// @example {}
func (s *NotesService) List(ctx context.Context, req *ListNotesRequest, rsp *ListNotesResponse) error {
	rsp.Notes = []string{"Install the micro CLI", "Run a service", "Chat with an agent"}
	fmt.Fprintln(s.w, "  [notes] listed starter notes")
	return nil
}

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
func (m *mockModel) String() string      { return "first-agent-mock" }
func (m *mockModel) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) {
	return nil, fmt.Errorf("stream not supported by first-agent mock")
}

func (m *mockModel) Generate(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	for _, tool := range req.Tools {
		if strings.Contains(tool.Name, "List") && m.opts.ToolHandler != nil {
			m.opts.ToolHandler(ctx, ai.ToolCall{ID: "list-notes", Name: tool.Name, Input: map[string]any{}})
			break
		}
	}
	return &ai.Response{Answer: "Your first agent read the notes service and found three steps: install the CLI, run a service, then chat with an agent."}, nil
}

func waitFor(reg registry.Registry, names ...string) error {
	deadline := time.Now().Add(5 * time.Second)
	for _, name := range names {
		for {
			if svcs, err := reg.GetService(name); err == nil && len(svcs) > 0 && len(svcs[0].Nodes) > 0 {
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timed out waiting for %s", name)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	return nil
}

func runFirstAgent() error {
	return runFirstAgentWithWriter(os.Stdout)
}

func runFirstAgentWithWriter(w io.Writer) error {
	ai.Register("first-agent-mock", newMock)

	reg := registry.NewMemoryRegistry()
	br := broker.NewMemoryBroker()
	if err := br.Init(); err != nil {
		return fmt.Errorf("init broker: %w", err)
	}
	if err := br.Connect(); err != nil {
		return fmt.Errorf("connect broker: %w", err)
	}
	defer br.Disconnect()
	cl := client.NewClient(client.Registry(reg), client.Selector(selector.NewSelector(selector.Registry(reg))), client.Broker(br))

	notes := service.New(service.Name("notes"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl), service.Broker(br), service.HandleSignal(false))
	if err := notes.Handle(&NotesService{w: w}); err != nil {
		return fmt.Errorf("handle notes: %w", err)
	}
	svcErr := make(chan error, 1)
	go func() { svcErr <- notes.Run() }()
	defer notes.Server().Stop()

	assistant := agent.New(
		agent.Name("assistant"),
		agent.Address("127.0.0.1:0"),
		agent.Services("notes"),
		agent.Prompt("You are a friendly first agent. Use the notes service before answering."),
		agent.Provider("first-agent-mock"),
		agent.WithRegistry(reg),
		agent.WithClient(cl),
		agent.WithBroker(br),
		agent.WithStore(store.NewMemoryStore()),
	)
	agentErr := make(chan error, 1)
	go func() { agentErr <- assistant.Run() }()
	defer assistant.Stop()

	if err := waitFor(reg, "notes", "assistant"); err != nil {
		select {
		case runErr := <-svcErr:
			return fmt.Errorf("run notes: %w", runErr)
		case runErr := <-agentErr:
			return fmt.Errorf("run assistant: %w", runErr)
		default:
		}
		return err
	}

	fmt.Fprintln(w, "First agent (provider: mock, no API key)")
	fmt.Fprintln(w, "> Summarize my next steps")
	resp, err := assistant.Ask(context.Background(), "Summarize my next steps")
	if err != nil {
		return fmt.Errorf("ask assistant: %w", err)
	}
	fmt.Fprintln(w, "assistant:", resp.Reply)
	fmt.Fprintln(w, "✓ service-backed agent completed without provider secrets")
	return nil
}

func main() {
	if err := runFirstAgent(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

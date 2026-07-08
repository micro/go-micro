// Package agent provides the Agent abstraction for Go Micro.
//
// An Agent is a service with an LLM inside it. It registers a Chat
// RPC endpoint, discovers its assigned services' tools, and
// orchestrates them intelligently.
//
//	agent := micro.NewAgent("task-mgr",
//	    micro.AgentServices("task"),
//	    micro.AgentPrompt("You manage tasks."),
//	    micro.AgentProvider("anthropic"),
//	)
//	agent.Run()
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/gateway/a2a"
	"go-micro.dev/v6/server"
	"go-micro.dev/v6/store"

	_ "go-micro.dev/v6/ai/anthropic"
	_ "go-micro.dev/v6/ai/atlascloud"
	_ "go-micro.dev/v6/ai/gemini"
	_ "go-micro.dev/v6/ai/groq"
	_ "go-micro.dev/v6/ai/mistral"
	_ "go-micro.dev/v6/ai/ollama"
	_ "go-micro.dev/v6/ai/openai"
	_ "go-micro.dev/v6/ai/together"
)

// Agent is the interface for an AI agent that manages services.
type Agent interface {
	Name() string
	Init(...Option)
	Options() Options
	Ask(ctx context.Context, message string) (*Response, error)
	Stream(ctx context.Context, message string) (ai.Stream, error)
	Run() error
	Stop() error
	String() string
}

// Response is what an agent returns from Chat.
type Response struct {
	Reply     string
	ToolCalls []ai.ToolCall
	Agent     string

	// RunID correlates this Ask with tool calls, trace spans, and the
	// persisted run timeline. ParentID is set when this response belongs
	// to a delegated sub-agent run.
	RunID    string
	ParentID string
}

type agentImpl struct {
	opts   Options
	model  ai.Model
	tools  *ai.Tools
	mem    Memory
	server server.Server
	mu     sync.Mutex

	// ephemeral marks a short-lived sub-agent created by delegation.
	// Ephemeral agents run with an isolated context: they load and
	// persist no history, and have no built-in tools (so they cannot
	// plan or re-delegate).
	ephemeral bool

	// steps counts tool executions in the current Ask, for MaxSteps.
	steps int
	// calls counts identical tool calls (name+args) in the current Ask,
	// for LoopLimit.
	calls map[string]int

	// runID correlates the tool calls of the current Ask; parentRunID is
	// the run that delegated to this one (set on ephemeral sub-agents).
	// Both are surfaced to tool wrappers via ai.RunInfo on the context.
	runID       string
	parentRunID string

	// pause records a guardrail approval pause raised during the current
	// Ask. The model provider only sees a refused tool result; the agent
	// converts it into a durable paused run instead of completing the run.
	pause *approvalPause

	// currentRun points at the checkpoint record for the Ask currently
	// holding mu. Tool execution updates it so resumed runs can reuse
	// completed tool results without replaying side effects.
	currentRun *flow.Run

	// delegateCalls collapses concurrent equivalent delegate tool calls so a
	// provider replay cannot fan out duplicate delegated side effects before the
	// durable delegate-result cache is written.
	delegateMu    sync.Mutex
	delegateCalls map[string]*delegateCall
}

// New creates a new Agent.
func New(opts ...Option) Agent {
	return &agentImpl{
		opts: newOptions(opts...),
	}
}

// newEphemeral creates a short-lived sub-agent for a delegated subtask.
// It shares the parent's provider, model, and infrastructure but runs
// with an isolated context: it loads and persists no history and has no
// built-in tools (so it can neither plan nor re-delegate). Returns the
// concrete type because ephemeral is an internal construction detail,
// not a public option.
func newEphemeral(opts ...Option) *agentImpl {
	return &agentImpl{
		opts:      newOptions(opts...),
		ephemeral: true,
	}
}

func (a *agentImpl) Name() string {
	return a.opts.Name
}

func (a *agentImpl) Init(opts ...Option) {
	for _, o := range opts {
		o(&a.opts)
	}
	a.setup()
}

func (a *agentImpl) Options() Options {
	return a.opts
}

func (a *agentImpl) String() string {
	return "agent"
}

func (a *agentImpl) setup() {
	a.setupWithToolHandler(nil)
}

func (a *agentImpl) setupWithToolHandler(handler ai.ToolHandler) {
	var modelOpts []ai.Option
	modelOpts = append(modelOpts, ai.WithAPIKey(a.opts.APIKey))
	if a.opts.Model != "" {
		modelOpts = append(modelOpts, ai.WithModel(a.opts.Model))
	}
	if a.opts.BaseURL != "" {
		modelOpts = append(modelOpts, ai.WithBaseURL(a.opts.BaseURL))
	}

	// Reuse the existing tools instance: its name map is populated by
	// discoverTools, and rebuilding it here would orphan a base handler that
	// already captured the old instance (breaking StreamAsk tool resolution).
	if a.tools == nil {
		a.tools = ai.NewTools(a.opts.Registry, ai.ToolClient(a.opts.Client))
	}
	if handler == nil {
		handler = a.toolHandler()
	}
	modelOpts = append(modelOpts, ai.WithToolHandler(handler))
	a.model = ai.New(a.opts.Provider, modelOpts...)
	if a.model != nil {
		a.model = a.tracedModel(a.model)
	}

	if a.mem != nil {
		return
	}

	// Memory is pluggable. Use the configured one, otherwise the default
	// store-backed memory — except ephemeral sub-agents, which keep an
	// isolated, non-persistent context.
	switch {
	case a.opts.Memory != nil:
		a.mem = a.opts.Memory
	case a.ephemeral:
		a.mem = NewInMemory(a.opts.HistoryLimit)
	case a.opts.MemoryCompaction.MaxMessages > 0:
		a.mem = NewCompactingMemoryWithOptions(a.stateStore(), "history", a.opts.MemoryCompaction)
	case a.opts.MemoryRetrievalLimit > 0:
		a.mem = NewRetrievalMemory(a.stateStore(), "history", a.opts.MemoryRetrievalLimit)
	default:
		a.mem = NewMemory(a.stateStore(), "history", a.opts.HistoryLimit)
	}
}

// stateStore returns the agent's own state store, scoped to its name so
// memory and plan live in their own table ("agent/{name}") rather than a
// shared global one. The scoped handle injects the database/table per
// operation without mutating the underlying store.
func (a *agentImpl) stateStore() store.Store {
	s := a.opts.Store
	if s == nil {
		s = store.DefaultStore
	}
	return store.Scope(s, "agent", a.opts.Name)
}

// Ask sends a message and returns the agent's response.
// This is the programmatic API for direct use.
func (a *agentImpl) Ask(ctx context.Context, message string) (*Response, error) {
	return a.ask(ctx, message, a.parentRunID)
}

// Stream sends a message and returns a streaming model response. Tool-calling
// agent runs still use Ask; Stream is for chat turns where immediate token
// delivery is more important than tool orchestration.
func (a *agentImpl) Stream(ctx context.Context, message string) (ai.Stream, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.model == nil {
		a.setup()
	}
	toolList, err := a.discoverTools()
	if err != nil {
		return nil, fmt.Errorf("discover tools: %w", err)
	}
	messages := append([]ai.Message(nil), a.mem.Messages()...)
	messages = append(messages, ai.Message{Role: "user", Content: message})
	stream, err := a.model.Stream(ctx, &ai.Request{
		Prompt:       message,
		SystemPrompt: a.buildPrompt(),
		Tools:        toolList,
		Messages:     messages,
	})
	if err != nil {
		return nil, err
	}
	a.mem.Add("user", message)
	return &memoryRecordingStream{stream: stream, memory: a.mem}, nil
}

// Pending returns checkpointed agent runs that have not completed. It mirrors
// flow.Pending for startup recovery loops that drain durable agent work.
func Pending(ctx context.Context, ag Agent) ([]flow.Run, error) {
	a, ok := ag.(*agentImpl)
	if !ok {
		return nil, fmt.Errorf("agent pending: unsupported agent implementation %T", ag)
	}
	return a.pending(ctx)
}

// ResumePending resumes every checkpointed agent run that has not completed
// yet, in the same oldest-first order returned by Pending.
//
// It is a convenience for service startup and recovery loops: after recreating
// an agent with the same checkpoint store, call ResumePending to drain the
// durable backlog without listing and resuming each run manually. If any run
// fails again, ResumePending stops and returns that run id with the error so
// callers can log, alert, or retry later without hiding the failing run.
func ResumePending(ctx context.Context, ag Agent) (string, error) {
	a, ok := ag.(*agentImpl)
	if !ok {
		return "", fmt.Errorf("agent resume pending: unsupported agent implementation %T", ag)
	}
	runs, err := a.pending(ctx)
	if err != nil {
		return "", err
	}
	for _, run := range runs {
		if _, err := a.resume(ctx, run.ID); err != nil {
			return run.ID, err
		}
	}
	return "", nil
}

func (a *agentImpl) ask(ctx context.Context, message, parentRunID string) (*Response, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.model == nil {
		a.setup()
	}

	return a.askLocked(ctx, uuid.New().String(), message, parentRunID, nil, true)
}

func (a *agentImpl) askLocked(ctx context.Context, runID, message, parentRunID string, existing *flow.Run, addUserMessage bool) (*Response, error) {
	toolList, err := a.discoverTools()
	if err != nil {
		return nil, fmt.Errorf("discover tools: %w", err)
	}

	if addUserMessage {
		a.mem.Add("user", message)
	}
	a.steps = 0
	a.calls = map[string]int{}
	a.pause = nil

	// Correlate this run's tool calls and surface lineage to wrappers.
	a.runID = runID
	ctx = ai.WithRunInfo(ctx, ai.RunInfo{
		RunID:    a.runID,
		ParentID: parentRunID,
		Agent:    a.opts.Name,
	})
	run := a.newCheckpointRun(runID, message, parentRunID, existing)
	a.currentRun = &run
	defer func() { a.currentRun = nil }()
	if err := a.saveRun(ctx, run); err != nil {
		return nil, err
	}
	ctx, endRun := a.startRun(ctx, message)
	if existing != nil {
		a.recordTimelineEvent(ctx, RunEvent{Time: time.Now(), RunID: runID, ParentID: parentRunID, Agent: a.opts.Name, Kind: "resume", Name: run.State.Stage})
	}
	defer func() { endRun(err) }()

	messages := a.mem.Messages()
	if recall, ok := a.mem.(MemoryRecall); ok && a.opts.MemoryRecallLimit > 0 {
		if recalled := recall.Recall(message, a.opts.MemoryRecallLimit); len(recalled) > 0 {
			messages = append([]ai.Message{{
				Role:    "system",
				Content: "Relevant recalled memory follows; use it as durable prior context without assuming the whole conversation was replayed.",
			}}, append(recalled, messages...)...)
		}
	}

	// Some providers satisfy a saved plan one outstanding item per turn,
	// especially when the final item delegates to another agent. Allow enough
	// continuations for the services → agents → workflows harness to complete
	// every planned side effect without weakening the final unfinished-plan guard.
	const maxPlanCompletionTurns = 6
	var resp *ai.Response
	for planCompletionTurn := 0; ; planCompletionTurn++ {
		resp, err = ai.GenerateWithRetry(ctx, a.model, &ai.Request{
			Prompt:       message,
			SystemPrompt: a.buildPrompt(),
			Tools:        toolList,
			Messages:     messages,
		}, ai.GeneratePolicy{
			Timeout:     a.opts.ModelTimeout,
			MaxAttempts: a.opts.ModelMaxAttempts,
			Backoff:     a.opts.ModelRetryBackoff,
		})
		if err != nil {
			run.Status = agentRunFailureStatus(err)
			err = agentOperationalError(err)
			if a.currentRun != nil {
				run.Steps = a.currentRun.Steps
			}
			if len(run.Steps) == 0 {
				run.Steps = []flow.StepRecord{{Name: agentAskStep}}
			}
			run.Steps[0].Status = run.Status
			run.Steps[0].Error = err.Error()
			_ = a.saveRun(ctx, run)
			return nil, err
		}
		if a.pause != nil && a.opts.Checkpoint != nil {
			run.Status = "paused"
			run.State.Stage = agentApprovalStep
			run.State.Data = []byte(message)
			if a.pause.Tool == toolHumanInput {
				run.State.Stage = agentInputStep
				_ = run.State.Set(inputPause{OriginalMessage: message, Prompt: a.pause.Message})
			}
			run.Steps[0].Status = "paused"
			run.Steps[0].Error = a.pause.Message
			run.Steps[0].Result = a.pause.Tool
			if err := a.saveRun(ctx, run); err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("agent run %s paused for approval: %s", run.ID, a.pause.Message)
		}

		if len(resp.ToolCalls) == 0 {
			if calls, answer, ok := a.executeTextToolCalls(ctx, resp.Reply, toolList); ok {
				resp.ToolCalls = calls
				if resp.Answer == "" {
					resp.Answer = answer
				}
				trimmedReply := strings.TrimSpace(resp.Reply)
				if strings.HasPrefix(trimmedReply, "{") || strings.HasPrefix(trimmedReply, "[") || strings.HasPrefix(trimmedReply, "```") {
					resp.Reply = ""
				}
			}
		} else if calls, answer, ok := a.executeAdditionalTextToolCalls(ctx, resp.Reply, toolList, resp.ToolCalls); ok {
			resp.ToolCalls = append(resp.ToolCalls, calls...)
			if answer != "" {
				if resp.Answer == "" {
					resp.Answer = answer
				} else {
					resp.Answer += "\n" + answer
				}
			}
		}

		if a.opts.Checkpoint != nil {
			if unfinished := a.unfinishedPlanSteps(); len(unfinished) > 0 && planCompletionTurn < maxPlanCompletionTurns {
				if resp.Reply != "" {
					a.mem.Add("assistant", resp.Reply)
				}
				if resp.Answer != "" {
					a.mem.Add("assistant", resp.Answer)
				}
				message = fmt.Sprintf("Continue the same run by calling the required tool(s) for the unfinished plan steps below. Do not repeat completed work, do not provide a final answer yet, and complete at least one unfinished step this turn if a matching tool is available. Unfinished plan steps: %s", strings.Join(unfinished, ", "))
				a.mem.Add("user", message)
				messages = a.mem.Messages()
				continue
			}
		}
		break
	}

	if resp.Reply != "" {
		a.mem.Add("assistant", resp.Reply)
	}
	if resp.Answer != "" {
		a.mem.Add("assistant", resp.Answer)
	}

	reply := resp.Reply
	if resp.Answer != "" {
		if reply != "" {
			reply += "\n\n"
		}
		reply += resp.Answer
	}

	res := &Response{
		Reply:     reply,
		ToolCalls: resp.ToolCalls,
		Agent:     a.opts.Name,
		RunID:     a.runID,
		ParentID:  parentRunID,
	}
	if a.opts.Checkpoint != nil {
		if unfinished := a.unfinishedPlanSteps(); len(unfinished) > 0 {
			err = fmt.Errorf("agent run %s has unfinished plan steps: %s", run.ID, strings.Join(unfinished, ", "))
			run.Status = "failed"
			run.State.Stage = agentAskStep
			run.State.Data = []byte(message)
			if a.currentRun != nil {
				run.Steps = a.currentRun.Steps
			}
			if len(run.Steps) == 0 {
				run.Steps = []flow.StepRecord{{Name: agentAskStep}}
			}
			run.Steps[0].Status = "failed"
			run.Steps[0].Error = err.Error()
			_ = a.saveRun(ctx, run)
			return nil, err
		}
	}
	run.Status = "done"
	run.State.Stage = ""
	if b, marshalErr := json.Marshal(res); marshalErr == nil {
		run.State.Data = b
	}
	if a.currentRun != nil {
		run.Steps = a.currentRun.Steps
	}
	if len(run.Steps) == 0 {
		run.Steps = []flow.StepRecord{{Name: agentAskStep}}
	}
	run.Steps[0].Status = "done"
	run.Steps[0].Attempts++
	run.Steps[0].Result = reply
	if err := a.saveRun(ctx, run); err != nil {
		return nil, err
	}
	return res, nil
}

// Chat implements the proto AgentHandler interface for RPC.
// @example {"message": "What tasks are overdue?"}
func (a *agentImpl) Chat(ctx context.Context, req *pb.ChatRequest, rsp *pb.ChatResponse) error {
	resp, err := a.ask(ctx, req.Message, req.ParentId)
	if err != nil {
		return err
	}
	rsp.Reply = resp.Reply
	rsp.Agent = resp.Agent
	rsp.RunId = resp.RunID
	rsp.ParentId = resp.ParentID
	for _, tc := range resp.ToolCalls {
		input, _ := json.Marshal(tc.Input)
		rsp.ToolCalls = append(rsp.ToolCalls, &pb.ToolCall{
			Id:     tc.ID,
			Name:   tc.Name,
			Input:  string(input),
			Result: tc.Result,
		})
	}
	return nil
}

// Run starts the agent as a service with a Chat RPC endpoint.
func (a *agentImpl) Run() error {
	if a.model == nil {
		a.setup()
	}

	serverOpts := []server.Option{
		server.Name(a.opts.Name),
		server.Address(a.opts.Address),
		server.Registry(a.opts.Registry),
		server.Metadata(map[string]string{
			"type":     "agent",
			"services": strings.Join(a.opts.Services, ","),
		}),
	}
	if a.opts.Broker != nil {
		serverOpts = append(serverOpts, server.Broker(a.opts.Broker))
	}
	a.server = server.NewServer(serverOpts...)

	_ = pb.RegisterAgentHandler(a.server, a)

	if err := a.server.Start(); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	fmt.Printf("Agent %s registered (manages: %s)\n", a.opts.Name, strings.Join(a.opts.Services, ", "))

	// Optionally serve the agent directly over the A2A protocol, calling
	// Ask in-process — no separate gateway needed to be queried by URL.
	if a.opts.A2AAddress != "" {
		card := a2a.Card(a.opts.Name, "http://localhost"+a.opts.A2AAddress, "", a.opts.Services)
		handler := a2a.NewAgentStreamHandler(card, func(ctx context.Context, text string) (string, error) {
			resp, err := a.Ask(ctx, text)
			if err != nil {
				return "", err
			}
			return resp.Reply, nil
		}, a.streamAskAI)
		go func() {
			if err := http.ListenAndServe(a.opts.A2AAddress, handler); err != nil {
				fmt.Printf("agent %s A2A server: %v\n", a.opts.Name, err)
			}
		}()
		fmt.Printf("Agent %s serving A2A on %s\n", a.opts.Name, a.opts.A2AAddress)
	}

	ch := make(chan struct{})
	<-ch
	return nil
}

func (a *agentImpl) Stop() error {
	if a.server != nil {
		return a.server.Stop()
	}
	return nil
}

func (a *agentImpl) discoverTools() ([]ai.Tool, error) {
	all, err := a.tools.Discover()
	if err != nil {
		return nil, err
	}

	var scoped []ai.Tool
	for _, t := range all {
		if strings.HasPrefix(t.OriginalName, a.opts.Name+".") {
			continue
		}
		if len(a.opts.Services) == 0 {
			scoped = append(scoped, t)
			continue
		}
		for _, svc := range a.opts.Services {
			if strings.HasPrefix(t.OriginalName, svc+".") {
				scoped = append(scoped, t)
				break
			}
		}
	}

	// Developer-registered custom tools (WithTool).
	for i := range a.opts.tools {
		scoped = append(scoped, a.opts.tools[i].def)
	}

	// Expose the agent's own capabilities (plan, delegate) as tools.
	// Ephemeral sub-agents don't get them.
	if !a.ephemeral {
		scoped = append(scoped, builtinTools()...)
	}
	return scoped, nil
}

func (a *agentImpl) buildPrompt() string {
	var base string
	switch {
	case a.opts.Prompt != "":
		base = a.opts.Prompt
	case len(a.opts.Services) > 0:
		base = fmt.Sprintf("You are the %s agent. You manage these services: %s. Use the available tools to fulfill requests.",
			a.opts.Name, strings.Join(a.opts.Services, ", "))
	default:
		base = fmt.Sprintf("You are the %s agent. Use the available tools to fulfill requests.", a.opts.Name)
	}

	// Keep the agent oriented: surface its saved plan, if any.
	if !a.ephemeral {
		if plan := a.loadPlan(); plan != "" {
			base += "\n\nYour current plan (update it with the plan tool as you make progress):\n" + plan
		}
	}
	return base
}

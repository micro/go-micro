// Package a2a provides an Agent2Agent (A2A) protocol gateway for Go Micro
// agents. It exposes every registered agent to the wider A2A ecosystem
// without any extra code on the agent: agents are discovered from the
// registry (the ones advertising type=agent), an Agent Card is generated
// for each from its registry metadata, and incoming A2A tasks are
// translated to the agent's existing Agent.Chat RPC.
//
// This is the agent-side analog of the MCP gateway: MCP exposes your
// services as tools, A2A exposes your agents as agents. Cards are derived
// from the registry the same way MCP tools are — there is nothing to
// publish.
//
// Example:
//
//	go a2a.Serve(a2a.Options{
//		Registry: service.Options().Registry,
//		Address:  ":4000",
//		BaseURL:  "https://agents.example.com",
//	})
//
// Scope of this version: the JSON-RPC binding — `message/send`
// (returns a completed Task), `message/stream` (SSE with the completed
// Task event), `tasks/get`, multi-turn task continuation, push
// notification delivery, input-required handoffs, `tasks/resubscribe`,
// and Agent Card discovery.
package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/client"
	codecbytes "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/registry"
)

// protocolVersion is the A2A spec version this gateway targets. Verify
// against the current spec when upgrading.
const protocolVersion = "0.3.0"

// maxTasks bounds the in-memory task history retained for tasks/get.
const maxTasks = 1024

// Options configures the A2A gateway.
type Options struct {
	// Registry for discovering agents (required).
	Registry registry.Registry
	// Address to listen on (e.g. ":4000"). Used by Serve.
	Address string
	// BaseURL is the public base URL clients reach this gateway at, used
	// to build each Agent Card's `url`. Defaults to http://localhost<Address>.
	BaseURL string
	// Client for the Agent.Chat RPC (defaults to client.DefaultClient).
	Client client.Client
	// Logger for startup/debug output (defaults to log.Default()).
	Logger *log.Logger
}

// Gateway serves the A2A protocol over HTTP for the registry's agents.
type Gateway struct {
	opts Options
	disp *dispatcher
}

// New creates an A2A gateway.
func New(opts Options) *Gateway {
	if opts.Client == nil {
		opts.Client = client.DefaultClient
	}
	if opts.Registry == nil {
		opts.Registry = registry.DefaultRegistry
	}
	if opts.Logger == nil {
		opts.Logger = log.Default()
	}
	if opts.BaseURL == "" {
		opts.BaseURL = "http://localhost" + opts.Address
	}
	opts.BaseURL = strings.TrimRight(opts.BaseURL, "/")
	return &Gateway{opts: opts, disp: newDispatcher()}
}

// Invoke runs an agent for one message and returns its reply. It is the
// seam between the A2A protocol and however the agent is reached — an RPC
// to Agent.Chat (the gateway) or an in-process Ask (an embedded agent).
type Invoke func(ctx context.Context, text string) (string, error)

// StreamInvoke runs an agent for one message and returns streaming output chunks.
type StreamInvoke func(ctx context.Context, text string) (ai.Stream, error)

// NewAgentHandler returns an http.Handler that serves the A2A protocol
// for a single agent: its Agent Card at / and /.well-known/agent.json,
// and the JSON-RPC endpoint at /. invoke runs the agent. This is what an
// agent embeds to speak A2A directly, without a separate gateway.
func NewAgentHandler(card AgentCard, invoke Invoke) http.Handler {
	d := newDispatcher()
	mux := http.NewServeMux()
	card.URL = strings.TrimRight(card.URL, "/")
	serveCard := func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, card) }
	mux.HandleFunc("GET /{$}", serveCard)
	mux.HandleFunc("GET /.well-known/agent.json", serveCard)
	mux.HandleFunc("POST /{$}", func(w http.ResponseWriter, r *http.Request) { d.serve(w, r, invoke) })
	return mux
}

// NewAgentStreamHandler is like NewAgentHandler, but serves A2A message/stream
// by forwarding model chunks as server-sent task updates when stream is non-nil.
func NewAgentStreamHandler(card AgentCard, invoke Invoke, stream StreamInvoke) http.Handler {
	d := newDispatcher()
	mux := http.NewServeMux()
	card.URL = strings.TrimRight(card.URL, "/")
	serveCard := func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, card) }
	mux.HandleFunc("GET /{$}", serveCard)
	mux.HandleFunc("GET /.well-known/agent.json", serveCard)
	mux.HandleFunc("POST /{$}", func(w http.ResponseWriter, r *http.Request) { d.serveWithStream(w, r, invoke, stream) })
	return mux
}

// Serve creates a gateway and serves it on opts.Address (blocking).
func Serve(opts Options) error {
	g := New(opts)
	g.opts.Logger.Printf("[a2a] gateway listening on %s (base %s)", opts.Address, g.opts.BaseURL)
	return http.ListenAndServe(opts.Address, g.Handler())
}

// Handler returns the gateway's HTTP handler.
func (g *Gateway) Handler() http.Handler {
	mux := http.NewServeMux()
	// Discovery: a directory of all agent cards.
	mux.HandleFunc("GET /agents", g.handleList)
	// Per-agent card (served at the agent's url and at its well-known path).
	mux.HandleFunc("GET /agents/{name}", g.handleCard)
	mux.HandleFunc("GET /agents/{name}/.well-known/agent.json", g.handleCard)
	// Per-agent JSON-RPC endpoint.
	mux.HandleFunc("POST /agents/{name}", g.handleRPC)
	// Top-level well-known: serve the single agent's card if there's
	// exactly one, otherwise point to the directory.
	mux.HandleFunc("GET /.well-known/agent.json", g.handleWellKnown)
	return mux
}

// ---------------------------------------------------------------------------
// A2A types (JSON-RPC binding)
// ---------------------------------------------------------------------------

// AgentCard describes an agent for discovery.
type AgentCard struct {
	Name               string       `json:"name"`
	Description        string       `json:"description,omitempty"`
	URL                string       `json:"url"`
	Version            string       `json:"version"`
	ProtocolVersion    string       `json:"protocolVersion"`
	Provider           *Provider    `json:"provider,omitempty"`
	Capabilities       Capabilities `json:"capabilities"`
	DefaultInputModes  []string     `json:"defaultInputModes"`
	DefaultOutputModes []string     `json:"defaultOutputModes"`
	Skills             []Skill      `json:"skills"`
}

// Provider identifies the organization behind an agent.
type Provider struct {
	Organization string `json:"organization"`
	URL          string `json:"url,omitempty"`
}

// Capabilities declares optional A2A features the agent supports.
type Capabilities struct {
	Streaming         bool `json:"streaming"`
	PushNotifications bool `json:"pushNotifications"`
}

// Skill is a capability advertised on the Agent Card.
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// Part is one piece of message/artifact content. This gateway handles text.
type Part struct {
	Kind string `json:"kind"` // "text"
	Text string `json:"text,omitempty"`
}

// Message is a turn in an A2A conversation.
type Message struct {
	Role      string `json:"role"` // "user" | "agent"
	Parts     []Part `json:"parts"`
	MessageID string `json:"messageId,omitempty"`
	TaskID    string `json:"taskId,omitempty"`
	ContextID string `json:"contextId,omitempty"`
	Kind      string `json:"kind"` // "message"
}

// TaskStatus is a task's lifecycle state.
type TaskStatus struct {
	State     string `json:"state"`
	Timestamp string `json:"timestamp"`
}

// Artifact is an output produced by a task.
type Artifact struct {
	ArtifactID string `json:"artifactId"`
	Parts      []Part `json:"parts"`
}

// Task is the unit of work returned by message/send and tasks/get.
type Task struct {
	ID        string     `json:"id"`
	ContextID string     `json:"contextId"`
	Status    TaskStatus `json:"status"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
	History   []Message  `json:"history,omitempty"`
	Kind      string     `json:"kind"` // "task"
}

// PushNotificationConfig tells the gateway where to POST task updates for a
// task. The gateway stores one config per task and delivers best-effort JSON
// task snapshots whenever that task changes.
type PushNotificationConfig struct {
	URL            string            `json:"url"`
	Token          string            `json:"token,omitempty"`
	Authentication map[string]string `json:"authentication,omitempty"`
}

// Task states (JSON-RPC binding wire values).
const (
	stateCompleted     = "completed"
	stateFailed        = "failed"
	stateWorking       = "working"
	stateInputRequired = "input-required"
)

// JSON-RPC envelopes.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// JSON-RPC error codes (standard + A2A-specific).
const (
	errParse          = -32700
	errInvalidRequest = -32600
	errMethodNotFound = -32601
	errInvalidParams  = -32602
	errInternal       = -32603
	errTaskNotFound   = -32001
	errNotCancelable  = -32002
)

// ---------------------------------------------------------------------------
// discovery — cards generated from the registry
// ---------------------------------------------------------------------------

// agents returns the registered agents (services advertising type=agent),
// as a name->card map.
func (g *Gateway) cards() ([]AgentCard, error) {
	svcs, err := g.opts.Registry.ListServices()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []AgentCard
	for _, s := range svcs {
		if seen[s.Name] {
			continue
		}
		recs, err := g.opts.Registry.GetService(s.Name)
		if err != nil || len(recs) == 0 {
			continue
		}
		meta := agentMetadata(recs[0])
		if meta == nil {
			continue
		}
		seen[s.Name] = true
		out = append(out, g.card(s.Name, meta))
	}
	return out, nil
}

// agentMetadata returns the metadata of a service iff it is an agent.
func agentMetadata(svc *registry.Service) map[string]string {
	if svc.Metadata != nil && svc.Metadata["type"] == "agent" {
		return svc.Metadata
	}
	for _, n := range svc.Nodes {
		if n.Metadata != nil && n.Metadata["type"] == "agent" {
			return n.Metadata
		}
	}
	return nil
}

// card builds an Agent Card for a named agent from its registry metadata.
func (g *Gateway) card(name string, meta map[string]string) AgentCard {
	var services []string
	if meta["services"] != "" {
		services = strings.Split(meta["services"], ",")
	}
	return Card(name, g.opts.BaseURL+"/agents/"+name, meta["description"], services)
}

// Card builds an Agent Card for an agent. url is the agent's A2A endpoint
// (the card's `url`); description defaults from the services it manages.
// Agents embedding the A2A handler use this to build their own card.
func Card(name, url, description string, services []string) AgentCard {
	if description == "" {
		if len(services) > 0 {
			description = fmt.Sprintf("Go Micro agent managing: %s", strings.Join(services, ","))
		} else {
			description = "Go Micro agent"
		}
	}
	return AgentCard{
		Name:            name,
		Description:     description,
		URL:             url,
		Version:         "1.0.0",
		ProtocolVersion: protocolVersion,
		Capabilities:    Capabilities{Streaming: true, PushNotifications: true},
		// The agent converses over a single Chat endpoint; advertise that
		// as one skill, tagged with the services it manages.
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Skills: []Skill{{
			ID:          "chat",
			Name:        "Chat",
			Description: "Converse with the agent to operate its services.",
			Tags:        services,
		}},
	}
}

// lookupCard returns the card for a single agent by name.
func (g *Gateway) lookupCard(name string) (AgentCard, bool) {
	recs, err := g.opts.Registry.GetService(name)
	if err != nil || len(recs) == 0 {
		return AgentCard{}, false
	}
	meta := agentMetadata(recs[0])
	if meta == nil {
		return AgentCard{}, false
	}
	return g.card(name, meta), true
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

func (g *Gateway) handleList(w http.ResponseWriter, _ *http.Request) {
	cards, err := g.cards()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": cards})
}

func (g *Gateway) handleCard(w http.ResponseWriter, r *http.Request) {
	card, ok := g.lookupCard(r.PathValue("name"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, card)
}

func (g *Gateway) handleWellKnown(w http.ResponseWriter, r *http.Request) {
	cards, err := g.cards()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(cards) == 1 {
		writeJSON(w, http.StatusOK, cards[0])
		return
	}
	// More than one (or zero) agent: there's no single card here.
	writeJSON(w, http.StatusNotFound, map[string]any{
		"error":     "multiple or no agents; fetch a specific card",
		"directory": g.opts.BaseURL + "/agents",
	})
}

func (g *Gateway) handleRPC(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, ok := g.lookupCard(name); !ok {
		writeRPC(w, nil, nil, &rpcError{Code: errInvalidParams, Message: "unknown agent: " + name})
		return
	}
	g.disp.serve(w, r, func(ctx context.Context, text string) (string, error) {
		return g.callAgent(ctx, name, text)
	})
}

// dispatcher handles A2A JSON-RPC requests against an Invoke function and
// retains recent tasks for tasks/get. It is shared by the gateway (one
// per registry) and embedded agents (one per agent).
type dispatcher struct {
	mu          sync.Mutex
	tasks       map[string]*Task
	pushConfigs map[string]PushNotificationConfig
	watchers    map[string]map[chan *Task]struct{}
	order       []string // task ids in insertion order, for bounded eviction
}

func newDispatcher() *dispatcher {
	return &dispatcher{tasks: map[string]*Task{}, pushConfigs: map[string]PushNotificationConfig{}, watchers: map[string]map[chan *Task]struct{}{}}
}

func (d *dispatcher) serve(w http.ResponseWriter, r *http.Request, invoke Invoke) {
	d.serveWithStream(w, r, invoke, nil)
}

func (d *dispatcher) serveWithStream(w http.ResponseWriter, r *http.Request, invoke Invoke, streamInvoke StreamInvoke) {
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPC(w, nil, nil, &rpcError{Code: errParse, Message: "parse error"})
		return
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		writeRPC(w, req.ID, nil, &rpcError{Code: errInvalidRequest, Message: "invalid request"})
		return
	}

	switch req.Method {
	case "message/send":
		d.send(requestContext(r.Context()), w, req, invoke)
	case "message/stream":
		if streamInvoke != nil {
			d.streamChunks(requestContext(r.Context()), w, req, streamInvoke)
			return
		}
		d.stream(requestContext(r.Context()), w, req, invoke)
	case "tasks/get":
		d.get(w, req)
	case "tasks/pushNotificationConfig/set":
		d.setPushConfig(w, req)
	case "tasks/pushNotificationConfig/get":
		d.getPushConfig(w, req)
	case "tasks/cancel":
		// v1 tasks complete synchronously, so they're already terminal.
		writeRPC(w, req.ID, nil, &rpcError{Code: errNotCancelable, Message: "task is not cancelable"})
	case "tasks/resubscribe":
		d.resubscribe(requestContext(r.Context()), w, req)
	default:
		writeRPC(w, req.ID, nil, &rpcError{Code: errMethodNotFound, Message: "method not found: " + req.Method})
	}
}

type sendParams struct {
	Message Message `json:"message"`
}

func (d *dispatcher) send(ctx context.Context, w http.ResponseWriter, req rpcRequest, invoke Invoke) {
	task, e := d.run(ctx, req.Params, invoke)
	if e != nil {
		writeRPC(w, req.ID, nil, e)
		return
	}
	writeRPC(w, req.ID, task, nil)
}

func (d *dispatcher) stream(ctx context.Context, w http.ResponseWriter, req rpcRequest, invoke Invoke) {
	task, e := d.run(ctx, req.Params, invoke)
	if e != nil {
		writeRPC(w, req.ID, nil, e)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(sseWriter{w: w}).Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: task})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (d *dispatcher) streamChunks(ctx context.Context, w http.ResponseWriter, req rpcRequest, invoke StreamInvoke) {
	var p sendParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		writeRPC(w, req.ID, nil, &rpcError{Code: errInvalidParams, Message: "invalid params"})
		return
	}
	text := textOf(p.Message.Parts)
	if text == "" {
		writeRPC(w, req.ID, nil, &rpcError{Code: errInvalidParams, Message: "message has no text part"})
		return
	}
	stream, err := invoke(ctx, text)
	if err != nil {
		writeRPC(w, req.ID, nil, &rpcError{Code: errInternal, Message: err.Error()})
		return
	}
	defer stream.Close()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(sseWriter{w: w})
	flush := func() {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
	taskID := uuid.New().String()
	contextID := p.Message.ContextID
	if contextID == "" {
		contextID = uuid.New().String()
	}
	var reply strings.Builder
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			task := taskFromReplyWithIDs(p.Message, reply.String(), stateCompleted, taskID, contextID)
			d.store(task)
			_ = enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: task})
			flush()
			return
		}
		if err != nil {
			task := taskFromReplyWithIDs(p.Message, "error: "+err.Error(), stateFailed, taskID, contextID)
			d.store(task)
			_ = enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: task, Error: &rpcError{Code: errInternal, Message: err.Error()}})
			flush()
			return
		}
		if chunk == nil || chunk.Reply == "" {
			continue
		}
		reply.WriteString(chunk.Reply)
		task := taskFromReplyWithIDs(p.Message, reply.String(), stateWorking, taskID, contextID)
		d.store(task)
		_ = enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: task})
		flush()
	}
}

func (d *dispatcher) run(ctx context.Context, params json.RawMessage, invoke Invoke) (*Task, *rpcError) {
	var p sendParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: errInvalidParams, Message: "invalid params"}
	}
	text := textOf(p.Message.Parts)
	if text == "" {
		return nil, &rpcError{Code: errInvalidParams, Message: "message has no text part"}
	}

	reply, err := invoke(ctx, text)
	state := stateCompleted
	if err != nil {
		reply = "error: " + err.Error()
		state = stateFailed
		if isInputRequiredError(err) {
			reply = err.Error()
			state = stateInputRequired
		}
	}
	task := d.taskFromReply(p.Message, reply, state)
	d.store(task)
	return task, nil
}

type getParams struct {
	ID string `json:"id"`
}

func (d *dispatcher) resubscribe(ctx context.Context, w http.ResponseWriter, req rpcRequest) {
	var p getParams
	if err := json.Unmarshal(req.Params, &p); err != nil || p.ID == "" {
		writeRPC(w, req.ID, nil, &rpcError{Code: errInvalidParams, Message: "invalid params"})
		return
	}
	ch, task, unsubscribe := d.subscribe(p.ID)
	if task == nil {
		writeRPC(w, req.ID, nil, &rpcError{Code: errTaskNotFound, Message: "task not found"})
		return
	}
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(sseWriter{w: w})
	flush := func() {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
	writeEvent := func(t *Task) bool {
		_ = enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: t})
		flush()
		return isTerminal(t.Status.State)
	}
	if writeEvent(task) {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case next := <-ch:
			if writeEvent(next) {
				return
			}
		}
	}
}

func (d *dispatcher) get(w http.ResponseWriter, req rpcRequest) {
	var p getParams
	if err := json.Unmarshal(req.Params, &p); err != nil || p.ID == "" {
		writeRPC(w, req.ID, nil, &rpcError{Code: errInvalidParams, Message: "invalid params"})
		return
	}
	d.mu.Lock()
	task := d.tasks[p.ID]
	d.mu.Unlock()
	if task == nil {
		writeRPC(w, req.ID, nil, &rpcError{Code: errTaskNotFound, Message: "task not found"})
		return
	}
	writeRPC(w, req.ID, task, nil)
}

type pushConfigParams struct {
	ID                     string                 `json:"id"`
	PushNotificationConfig PushNotificationConfig `json:"pushNotificationConfig"`
}

func (d *dispatcher) setPushConfig(w http.ResponseWriter, req rpcRequest) {
	var p pushConfigParams
	if err := json.Unmarshal(req.Params, &p); err != nil || p.ID == "" || p.PushNotificationConfig.URL == "" {
		writeRPC(w, req.ID, nil, &rpcError{Code: errInvalidParams, Message: "invalid params"})
		return
	}
	d.mu.Lock()
	task := d.tasks[p.ID]
	if task != nil {
		d.pushConfigs[p.ID] = p.PushNotificationConfig
	}
	d.mu.Unlock()
	if task == nil {
		writeRPC(w, req.ID, nil, &rpcError{Code: errTaskNotFound, Message: "task not found"})
		return
	}
	writeRPC(w, req.ID, map[string]any{"id": p.ID, "pushNotificationConfig": p.PushNotificationConfig}, nil)
	go d.deliverPush(p.ID, task)
}

func (d *dispatcher) getPushConfig(w http.ResponseWriter, req rpcRequest) {
	var p getParams
	if err := json.Unmarshal(req.Params, &p); err != nil || p.ID == "" {
		writeRPC(w, req.ID, nil, &rpcError{Code: errInvalidParams, Message: "invalid params"})
		return
	}
	d.mu.Lock()
	cfg, ok := d.pushConfigs[p.ID]
	d.mu.Unlock()
	if !ok {
		writeRPC(w, req.ID, nil, &rpcError{Code: errTaskNotFound, Message: "push notification config not found"})
		return
	}
	writeRPC(w, req.ID, map[string]any{"id": p.ID, "pushNotificationConfig": cfg}, nil)
}

// ---------------------------------------------------------------------------
// agent RPC
// ---------------------------------------------------------------------------

// callAgent invokes an agent's Agent.Chat endpoint over RPC and returns
// its reply — the same call the delegate tool and flows use.
func (g *Gateway) callAgent(ctx context.Context, name, message string) (string, error) {
	body, _ := json.Marshal(map[string]string{"message": message})
	req := g.opts.Client.NewRequest(name, "Agent.Chat", &codecbytes.Frame{Data: body})
	var rsp codecbytes.Frame
	if err := g.opts.Client.Call(ctx, req, &rsp); err != nil {
		return "", err
	}
	var out struct {
		Reply string `json:"reply"`
	}
	if err := json.Unmarshal(rsp.Data, &out); err != nil {
		return "", err
	}
	return out.Reply, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (d *dispatcher) store(t *Task) {
	d.mu.Lock()
	_, exists := d.tasks[t.ID]
	d.tasks[t.ID] = t
	if !exists {
		d.order = append(d.order, t.ID)
	}
	for len(d.order) > maxTasks {
		oldest := d.order[0]
		d.order = d.order[1:]
		delete(d.tasks, oldest)
		delete(d.pushConfigs, oldest)
	}
	for ch := range d.watchers[t.ID] {
		select {
		case ch <- t:
		default:
		}
	}
	d.mu.Unlock()
	go d.deliverPush(t.ID, t)
}

func (d *dispatcher) subscribe(taskID string) (chan *Task, *Task, func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	task := d.tasks[taskID]
	if task == nil {
		return nil, nil, func() {}
	}
	ch := make(chan *Task, 8)
	if d.watchers[taskID] == nil {
		d.watchers[taskID] = map[chan *Task]struct{}{}
	}
	d.watchers[taskID][ch] = struct{}{}
	return ch, task, func() {
		d.mu.Lock()
		delete(d.watchers[taskID], ch)
		if len(d.watchers[taskID]) == 0 {
			delete(d.watchers, taskID)
		}
		close(ch)
		d.mu.Unlock()
	}
}

func isTerminal(state string) bool {
	return state == stateCompleted || state == stateFailed || state == stateInputRequired
}

func isInputRequiredError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "input-required") || strings.Contains(msg, "input required") || strings.Contains(msg, "paused for approval")
}

func (d *dispatcher) taskFromReply(input Message, reply, state string) *Task {
	contextID := input.ContextID
	taskID := input.TaskID
	var history []Message
	if taskID != "" {
		d.mu.Lock()
		prev := d.tasks[taskID]
		if prev != nil {
			contextID = prev.ContextID
			history = append(history, prev.History...)
		}
		d.mu.Unlock()
	}
	if taskID == "" {
		taskID = uuid.New().String()
	}
	if contextID == "" {
		contextID = uuid.New().String()
	}
	return taskFromReplyWithIDsAndHistory(input, reply, state, taskID, contextID, history)
}

func taskFromReplyWithIDs(input Message, reply, state, taskID, contextID string) *Task {
	return taskFromReplyWithIDsAndHistory(input, reply, state, taskID, contextID, nil)
}

func taskFromReplyWithIDsAndHistory(input Message, reply, state, taskID, contextID string, history []Message) *Task {
	input.TaskID = taskID
	input.ContextID = contextID
	if input.Kind == "" {
		input.Kind = "message"
	}
	task := &Task{
		ID:        taskID,
		ContextID: contextID,
		Kind:      "task",
		History:   append(append([]Message{}, history...), input),
		Status:    TaskStatus{State: state, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		Artifacts: []Artifact{textArtifact(reply)},
	}
	task.History = append(task.History, Message{
		Role:      "agent",
		Parts:     []Part{{Kind: "text", Text: reply}},
		MessageID: uuid.New().String(),
		TaskID:    task.ID,
		ContextID: task.ContextID,
		Kind:      "message",
	})
	return task
}

func (d *dispatcher) deliverPush(taskID string, task *Task) {
	d.mu.Lock()
	cfg, ok := d.pushConfigs[taskID]
	d.mu.Unlock()
	if !ok || cfg.URL == "" || task == nil {
		return
	}
	body, err := json.Marshal(task)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, strings.NewReader(string(body)))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

func textOf(parts []Part) string {
	var b strings.Builder
	for _, p := range parts {
		if p.Kind == "text" || p.Kind == "" {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func textArtifact(text string) Artifact {
	return Artifact{
		ArtifactID: uuid.New().String(),
		Parts:      []Part{{Kind: "text", Text: text}},
	}
}

// requestContext carries request cancellation and deadlines into the downstream
// agent call without leaking HTTP transport context values into the go-micro
// client stack.
func requestContext(parent context.Context) context.Context {
	if err := parent.Err(); err != nil {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}
	ctx := context.Background()
	var cancel context.CancelFunc
	if deadline, ok := parent.Deadline(); ok {
		ctx, cancel = context.WithDeadline(ctx, deadline)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	go func() {
		<-parent.Done()
		cancel()
	}()
	return ctx
}

type sseWriter struct {
	w http.ResponseWriter
}

func (s sseWriter) Write(p []byte) (int, error) {
	if _, err := s.w.Write([]byte("data: ")); err != nil {
		return 0, err
	}
	n, err := s.w.Write(p)
	if err != nil {
		return n, err
	}
	if _, err := s.w.Write([]byte("\n")); err != nil {
		return n, err
	}
	return n, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeRPC(w http.ResponseWriter, id json.RawMessage, result any, e *rpcError) {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	writeJSON(w, http.StatusOK, rpcResponse{JSONRPC: "2.0", ID: id, Result: result, Error: e})
}

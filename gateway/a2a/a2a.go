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
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	// AllowPushURL authorizes an outbound push-notification callback URL
	// (tasks/pushNotificationConfig/set). Return a non-nil error to reject it.
	// When nil, a default SSRF-safe policy applies: only http/https URLs whose
	// host does not resolve to a loopback, private, link-local, or unspecified
	// address are allowed, and the connection is pinned to that check at dial
	// time (DNS-rebinding safe). Set this to permit a trusted in-cluster
	// receiver, or to narrow delivery to an allowlist.
	AllowPushURL func(*url.URL) error
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
	g := &Gateway{opts: opts, disp: newDispatcher()}
	if opts.AllowPushURL != nil {
		// Operator owns the trust decision: use their policy and skip the
		// built-in private-IP dial guard so trusted in-cluster hosts resolve.
		g.disp.allowPushURL = opts.AllowPushURL
		g.disp.guardPushDial = false
	}
	return g
}

// Invoke runs an agent for one message and returns its reply. It is the
// seam between the A2A protocol and however the agent is reached — an RPC
// to Agent.Chat (the gateway) or an in-process Ask (an embedded agent).
type Invoke func(ctx context.Context, text string) (string, error)

// StreamInvoke runs an agent for one message and returns streaming output chunks.
type StreamInvoke func(ctx context.Context, text string) (ai.Stream, error)

// AgentHandlerOption configures an embedded A2A agent handler.
type AgentHandlerOption func(*dispatcher)

// WithPushURLPolicy sets the push-notification callback URL policy for an
// embedded agent handler (the analog of Options.AllowPushURL on the gateway).
// Return a non-nil error to reject a URL. Without it, the default SSRF-safe
// policy applies. Supplying a policy also disables the built-in private-IP dial
// guard, so a trusted in-cluster receiver resolves.
func WithPushURLPolicy(allow func(*url.URL) error) AgentHandlerOption {
	return func(d *dispatcher) {
		if allow == nil {
			return
		}
		d.allowPushURL = allow
		d.guardPushDial = false
	}
}

// NewAgentHandler returns an http.Handler that serves the A2A protocol
// for a single agent: its Agent Card at / and /.well-known/agent.json,
// and the JSON-RPC endpoint at /. invoke runs the agent. This is what an
// agent embeds to speak A2A directly, without a separate gateway.
func NewAgentHandler(card AgentCard, invoke Invoke, opts ...AgentHandlerOption) http.Handler {
	d := newDispatcher()
	for _, o := range opts {
		o(d)
	}
	mux := http.NewServeMux()
	card.URL = strings.TrimRight(card.URL, "/")
	serveCard := func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, card) }
	mux.HandleFunc("GET /{$}", serveCard)
	// A2A 0.3.0 discovery is /.well-known/agent-card.json; agent.json is the
	// pre-0.3 alias, kept so existing clients don't break.
	mux.HandleFunc("GET /.well-known/agent-card.json", serveCard)
	mux.HandleFunc("GET /.well-known/agent.json", serveCard)
	mux.HandleFunc("POST /{$}", func(w http.ResponseWriter, r *http.Request) { d.serve(w, r, invoke) })
	return mux
}

// NewAgentStreamHandler is like NewAgentHandler, but serves A2A message/stream
// by forwarding model chunks as server-sent task updates when stream is non-nil.
func NewAgentStreamHandler(card AgentCard, invoke Invoke, stream StreamInvoke, opts ...AgentHandlerOption) http.Handler {
	d := newDispatcher()
	for _, o := range opts {
		o(d)
	}
	mux := http.NewServeMux()
	card.URL = strings.TrimRight(card.URL, "/")
	serveCard := func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, card) }
	mux.HandleFunc("GET /{$}", serveCard)
	mux.HandleFunc("GET /.well-known/agent-card.json", serveCard)
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
	// A2A 0.3.0 uses agent-card.json; agent.json is the pre-0.3 alias.
	mux.HandleFunc("GET /agents/{name}", g.handleCard)
	mux.HandleFunc("GET /agents/{name}/.well-known/agent-card.json", g.handleCard)
	mux.HandleFunc("GET /agents/{name}/.well-known/agent.json", g.handleCard)
	mux.HandleFunc("GET /agents/{name}/skills/{skill}", g.handleSkillCard)
	mux.HandleFunc("GET /agents/{name}/skills/{skill}/.well-known/agent-card.json", g.handleSkillCard)
	mux.HandleFunc("GET /agents/{name}/skills/{skill}/.well-known/agent.json", g.handleSkillCard)
	// Per-agent JSON-RPC endpoint.
	mux.HandleFunc("POST /agents/{name}", g.handleRPC)
	mux.HandleFunc("POST /agents/{name}/skills/{skill}", g.handleSkillRPC)
	// Top-level well-known: serve the single agent's card if there's
	// exactly one, otherwise point to the directory.
	mux.HandleFunc("GET /.well-known/agent-card.json", g.handleWellKnown)
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
	TaskResubscribe   bool `json:"taskResubscribe"`
	InputRequired     bool `json:"inputRequired"`
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
	Role        string             `json:"role"` // "user" | "agent"
	Parts       []Part             `json:"parts"`
	MessageID   string             `json:"messageId,omitempty"`
	TaskID      string             `json:"taskId,omitempty"`
	ContextID   string             `json:"contextId,omitempty"`
	Kind        string             `json:"kind"` // "message"
	AP2Mandates []AP2SignedMandate `json:"ap2Mandates,omitempty"`
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

// TaskStatusUpdateEvent is an A2A streaming event reporting a change in a
// task's status. External SSE clients parse stream events by `kind` and stop
// on the event whose `final` is true — a full Task snapshot (which older
// versions emitted) carries neither, so strict clients never terminate.
type TaskStatusUpdateEvent struct {
	TaskID    string     `json:"taskId"`
	ContextID string     `json:"contextId"`
	Kind      string     `json:"kind"` // "status-update"
	Status    TaskStatus `json:"status"`
	Final     bool       `json:"final"`
}

// TaskArtifactUpdateEvent is an A2A streaming event carrying an artifact (or,
// with Append, one incremental chunk of one).
type TaskArtifactUpdateEvent struct {
	TaskID    string   `json:"taskId"`
	ContextID string   `json:"contextId"`
	Kind      string   `json:"kind"` // "artifact-update"
	Artifact  Artifact `json:"artifact"`
	Append    bool     `json:"append,omitempty"`
	LastChunk bool     `json:"lastChunk,omitempty"`
}

func statusUpdateEvent(t *Task, final bool) TaskStatusUpdateEvent {
	return TaskStatusUpdateEvent{
		TaskID:    t.ID,
		ContextID: t.ContextID,
		Kind:      "status-update",
		Status:    t.Status,
		Final:     final,
	}
}

// Task is the unit of work returned by message/send and tasks/get.
type Task struct {
	ID               string             `json:"id"`
	ContextID        string             `json:"contextId"`
	Status           TaskStatus         `json:"status"`
	Artifacts        []Artifact         `json:"artifacts,omitempty"`
	History          []Message          `json:"history,omitempty"`
	Kind             string             `json:"kind"` // "task"
	AP2Mandates      []AP2SignedMandate `json:"ap2Mandates,omitempty"`
	AP2Verifications []AP2Verification  `json:"ap2Verifications,omitempty"`
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
	skills := skillsFromServices(services)
	return AgentCard{
		Name:               name,
		Description:        description,
		URL:                url,
		Version:            "1.0.0",
		ProtocolVersion:    protocolVersion,
		Capabilities:       Capabilities{Streaming: true, PushNotifications: true, TaskResubscribe: true, InputRequired: true},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Skills:             skills,
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

func (g *Gateway) lookupSkillCard(name, skillID string) (AgentCard, Skill, bool) {
	card, ok := g.lookupCard(name)
	if !ok {
		return AgentCard{}, Skill{}, false
	}
	for _, skill := range card.Skills {
		if skill.ID == skillID {
			return card, skill, true
		}
	}
	return AgentCard{}, Skill{}, false
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

func (g *Gateway) handleSkillCard(w http.ResponseWriter, r *http.Request) {
	card, skill, ok := g.lookupSkillCard(r.PathValue("name"), r.PathValue("skill"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	card.URL = g.opts.BaseURL + "/agents/" + r.PathValue("name") + "/skills/" + skill.ID
	card.Skills = []Skill{skill}
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

func (g *Gateway) handleSkillRPC(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	_, skill, ok := g.lookupSkillCard(name, r.PathValue("skill"))
	if !ok {
		writeRPC(w, nil, nil, &rpcError{Code: errInvalidParams, Message: "unknown agent skill: " + name + "/" + r.PathValue("skill")})
		return
	}
	g.disp.serve(w, r, func(ctx context.Context, text string) (string, error) {
		return g.callAgent(ctx, name, skillPrompt(skill, text))
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

	// allowPushURL authorizes an outbound push-notification callback URL; nil
	// means the default SSRF-safe policy. guardPushDial applies the private-IP
	// dial guard (on unless an operator supplied a custom policy).
	allowPushURL  func(*url.URL) error
	guardPushDial bool
}

func newDispatcher() *dispatcher {
	return &dispatcher{
		tasks:         map[string]*Task{},
		pushConfigs:   map[string]PushNotificationConfig{},
		watchers:      map[string]map[chan *Task]struct{}{},
		allowPushURL:  defaultPushURLPolicy,
		guardPushDial: true,
	}
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
			d.streamChunks(requestContext(r.Context()), w, req, streamInvoke, invoke)
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
	enc, flush := sseResponse(w)
	// The Task snapshot first (carries ids and the final artifact), then a
	// terminal status-update so external SSE clients see `final:true` and stop.
	writeSSE(enc, flush, req.ID, task)
	writeSSE(enc, flush, req.ID, statusUpdateEvent(task, true))
}

func (d *dispatcher) streamChunks(ctx context.Context, w http.ResponseWriter, req rpcRequest, invoke StreamInvoke, fallback Invoke) {
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
		if errors.Is(err, ai.ErrStreamingUnsupported) && fallback != nil {
			d.stream(ctx, w, req, fallback)
			return
		}
		writeRPC(w, req.ID, nil, &rpcError{Code: errInternal, Message: err.Error()})
		return
	}
	defer stream.Close()
	enc, flush := sseResponse(w)
	taskID := uuid.New().String()
	contextID := p.Message.ContextID
	if contextID == "" {
		contextID = uuid.New().String()
	}
	// One artifact id for the whole stream so append:true chunks target it.
	artifactID := uuid.New().String()

	// Open with the Task snapshot (working) so the client learns the ids.
	initial := taskFromReplyWithIDs(p.Message, "", stateWorking, taskID, contextID)
	d.store(initial)
	writeSSE(enc, flush, req.ID, initial)

	var reply strings.Builder
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			task := taskFromReplyWithIDs(p.Message, reply.String(), stateCompleted, taskID, contextID)
			d.store(task)
			// Spec-shaped terminal: a status-update with final:true — not a
			// full Task snapshot, which carries no terminal marker.
			writeSSE(enc, flush, req.ID, statusUpdateEvent(task, true))
			return
		}
		if err != nil {
			task := taskFromReplyWithIDs(p.Message, "error: "+err.Error(), stateFailed, taskID, contextID)
			d.store(task)
			// A failed status-update (final) — never `result` and `error`
			// together in one response, which strict clients reject.
			writeSSE(enc, flush, req.ID, statusUpdateEvent(task, true))
			return
		}
		if chunk == nil || chunk.Reply == "" {
			continue
		}
		reply.WriteString(chunk.Reply)
		// Emit the delta as an append artifact-update; keep the stored task
		// current for tasks/get and resubscribe watchers.
		d.store(taskFromReplyWithIDs(p.Message, reply.String(), stateWorking, taskID, contextID))
		writeSSE(enc, flush, req.ID, TaskArtifactUpdateEvent{
			TaskID:    taskID,
			ContextID: contextID,
			Kind:      "artifact-update",
			Artifact:  Artifact{ArtifactID: artifactID, Parts: []Part{{Kind: "text", Text: chunk.Reply}}},
			Append:    true,
		})
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
	} else if strings.TrimSpace(reply) == "" {
		reply = "error: agent returned an empty response"
		state = stateFailed
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

	enc, flush := sseResponse(w)
	writeEvent := func(t *Task) bool {
		writeSSE(enc, flush, req.ID, t)
		if isTerminal(t.Status.State) {
			// Close the stream with a spec-shaped terminal marker so external
			// clients see `final:true`.
			writeSSE(enc, flush, req.ID, statusUpdateEvent(t, true))
			return true
		}
		return false
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
	// Reject SSRF-unsafe callback targets before storing them.
	if err := d.checkPushURL(p.PushNotificationConfig.URL); err != nil {
		writeRPC(w, req.ID, nil, &rpcError{Code: errInvalidParams, Message: "push notification url not allowed"})
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
	reply, err := decodeAgentChatReply(rsp.Data)
	if err != nil {
		return "", err
	}
	return reply, nil
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
		ID:          taskID,
		ContextID:   contextID,
		Kind:        "task",
		History:     append(append([]Message{}, history...), input),
		Status:      TaskStatus{State: state, Timestamp: time.Now().UTC().Format(time.RFC3339)},
		Artifacts:   []Artifact{textArtifact(reply)},
		AP2Mandates: append([]AP2SignedMandate{}, input.AP2Mandates...),
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
	// Defense in depth: re-validate the callback URL at delivery time in case
	// the policy tightened or the config was set before it applied.
	if err := d.checkPushURL(cfg.URL); err != nil {
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
	resp, err := d.pushClient().Do(req)
	if err == nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

func skillsFromServices(services []string) []Skill {
	if len(services) == 0 {
		return []Skill{{ID: "chat", Name: "Chat", Description: "Converse with the agent to operate its services."}}
	}
	seen := map[string]bool{}
	var skills []Skill
	for _, service := range services {
		service = strings.TrimSpace(service)
		if service == "" {
			continue
		}
		id := skillID(service)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		skills = append(skills, Skill{
			ID:          id,
			Name:        skillName(service),
			Description: fmt.Sprintf("Operate the %s service through this agent.", service),
			Tags:        []string{service},
		})
	}
	if len(skills) == 0 {
		return []Skill{{ID: "chat", Name: "Chat", Description: "Converse with the agent to operate its services."}}
	}
	return skills
}

func skillID(service string) string {
	service = strings.ToLower(strings.TrimSpace(service))
	var b strings.Builder
	dash := false
	for _, r := range service {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func skillName(service string) string {
	parts := strings.FieldsFunc(service, func(r rune) bool { return r == '-' || r == '_' || r == '.' || r == '/' || r == ' ' })
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func skillPrompt(skill Skill, text string) string {
	return fmt.Sprintf("Use the %q skill (%s) for this request.\n\n%s", skill.Name, skill.ID, text)
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

func decodeAgentChatReply(data []byte) (string, error) {
	var out struct {
		Reply   string `json:"reply"`
		Answer  string `json:"answer"`
		Content string `json:"content"`
		Text    string `json:"text"`
		Message struct {
			Content string `json:"content"`
			Text    string `json:"text"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", err
	}
	for _, candidate := range []string{
		out.Reply,
		out.Answer,
		out.Content,
		out.Text,
		out.Message.Content,
		out.Message.Text,
	} {
		if strings.TrimSpace(candidate) != "" {
			return candidate, nil
		}
	}
	return "", nil
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

// sseResponse writes the SSE response headers and returns an encoder and a
// flush func for emitting `data:`-framed JSON-RPC events.
func sseResponse(w http.ResponseWriter) (*json.Encoder, func()) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(sseWriter{w: w})
	return enc, func() {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

// writeSSE emits one JSON-RPC event (result only — never with an error) and flushes.
func writeSSE(enc *json.Encoder, flush func(), id json.RawMessage, result any) {
	_ = enc.Encode(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
	flush()
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

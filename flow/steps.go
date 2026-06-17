package flow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/client"
	codecbytes "go-micro.dev/v5/codec/bytes"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/store"
)

// State carries data across the steps of a flow run. It is a struct, not
// a map: Data is the serialized payload (set and read with Set/Scan), and
// Stage names the step the run is at — so you can always tell where it is,
// and the engine uses it as the resume point.
type State struct {
	Stage string `json:"stage"`
	Data  []byte `json:"data"`
}

// Set replaces the data with the JSON encoding of v.
func (s *State) Set(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.Data = b
	return nil
}

// Scan decodes the data into v (a pointer to the caller's struct).
func (s State) Scan(v any) error {
	if len(s.Data) == 0 {
		return nil
	}
	return json.Unmarshal(s.Data, v)
}

// String returns the data as a string, for text payloads.
func (s State) String() string { return string(s.Data) }

// StepFunc performs one step's work: it receives the carried state and
// returns the next state.
type StepFunc func(ctx context.Context, in State) (State, error)

// Step is one unit of a flow — a named action with an optional retry
// override. There is one Step kind; the action is the Run func, and the
// Call/LLM/Agent helpers produce the common ones.
type Step struct {
	Name  string
	Run   StepFunc
	Retry int // per-step override of the flow's retry (0 = use the flow default)
}

// StepRecord is the recorded outcome of one step within a run.
type StepRecord struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // pending | in_progress | done | failed
	Attempts int    `json:"attempts"`
	Result   string `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Run is the persisted record of one flow execution — what a Checkpoint
// saves and loads. It is retained for success and failure unless the flow
// opts into cleanup with DeleteOnSuccess.
type Run struct {
	ID      string       `json:"id"`
	Flow    string       `json:"flow"`
	State   State        `json:"state"`
	Steps   []StepRecord `json:"steps"`
	Status  string       `json:"status"` // running | done | failed
	Started time.Time    `json:"started"`
	Updated time.Time    `json:"updated"`
}

// Checkpoint persists and restores flow runs so a run survives a crash
// and resumes where it stopped. The built-in StoreCheckpoint is
// store-backed; implement this interface to plug in another durable
// execution backend.
type Checkpoint interface {
	Save(ctx context.Context, run Run) error
	Load(ctx context.Context, runID string) (Run, bool, error)
	Delete(ctx context.Context, runID string) error
	List(ctx context.Context) ([]Run, error)
}

type storeCheckpoint struct {
	store  store.Store
	prefix string // key namespace, e.g. "flow/checkout/runs/"
}

// StoreCheckpoint returns a store-backed Checkpoint whose run keys are
// namespaced under scope — pass the flow name so each flow's runs are
// kept apart (keys become "flow/{scope}/runs/{id}") rather than sharing
// one global keyspace. An empty scope falls back to "flow/runs/"; a nil
// store uses store.DefaultStore.
func StoreCheckpoint(s store.Store, scope string) Checkpoint {
	if s == nil {
		s = store.DefaultStore
	}
	prefix := "flow/runs/"
	if scope != "" {
		prefix = "flow/" + scope + "/runs/"
	}
	return &storeCheckpoint{store: s, prefix: prefix}
}

func (c *storeCheckpoint) Save(_ context.Context, run Run) error {
	run.Updated = time.Now()
	b, err := json.Marshal(run)
	if err != nil {
		return err
	}
	return c.store.Write(&store.Record{Key: c.prefix + run.ID, Value: b})
}

func (c *storeCheckpoint) Load(_ context.Context, runID string) (Run, bool, error) {
	recs, err := c.store.Read(c.prefix + runID)
	if err == store.ErrNotFound || len(recs) == 0 {
		return Run{}, false, nil
	}
	if err != nil {
		return Run{}, false, err
	}
	var run Run
	if err := json.Unmarshal(recs[0].Value, &run); err != nil {
		return Run{}, false, err
	}
	return run, true, nil
}

func (c *storeCheckpoint) Delete(_ context.Context, runID string) error {
	return c.store.Delete(c.prefix + runID)
}

func (c *storeCheckpoint) List(ctx context.Context) ([]Run, error) {
	keys, err := c.store.List(store.ListPrefix(c.prefix))
	if err != nil {
		return nil, err
	}
	var runs []Run
	for _, k := range keys {
		id := strings.TrimPrefix(k, c.prefix)
		if run, ok, err := c.Load(ctx, id); err == nil && ok {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

// defaultCheckpoint returns the configured checkpoint, or a store-backed
// default scoped to the flow name when the flow has steps (durable by
// default). Scoping by name keeps each flow's runs in their own keyspace
// rather than a global one.
func defaultCheckpoint(name string, o Options) Checkpoint {
	if o.Checkpoint != nil {
		return o.Checkpoint
	}
	if len(o.Steps) > 0 {
		return StoreCheckpoint(store.DefaultStore, name)
	}
	return nil
}

// runDeps are the flow resources the Call/LLM/Agent step helpers need.
// They are injected into the context for the duration of a run so a
// StepFunc keeps the clean (ctx, State) signature.
type runDeps struct {
	client client.Client
	model  ai.Model
	tools  *ai.Tools
}

type runCtxKey struct{}

func withDeps(ctx context.Context, d *runDeps) context.Context {
	return context.WithValue(ctx, runCtxKey{}, d)
}

func depsFrom(ctx context.Context) *runDeps {
	d, _ := ctx.Value(runCtxKey{}).(*runDeps)
	return d
}

// Call returns a StepFunc that invokes an RPC endpoint, sending the
// current state Data as the request body and storing the response as the
// new Data.
func Call(service, endpoint string) StepFunc {
	return func(ctx context.Context, in State) (State, error) {
		cl := client.DefaultClient
		if d := depsFrom(ctx); d != nil && d.client != nil {
			cl = d.client
		}
		body := in.Data
		if len(body) == 0 {
			body = []byte("{}")
		}
		req := cl.NewRequest(service, endpoint, &codecbytes.Frame{Data: body})
		var rsp codecbytes.Frame
		if err := cl.Call(ctx, req, &rsp); err != nil {
			return in, err
		}
		in.Data = rsp.Data
		return in, nil
	}
}

// Dispatch returns a StepFunc that hands the current state Data (as the
// message) to a registered agent's Agent.Chat endpoint and stores the
// reply as the new Data.
func Dispatch(name string) StepFunc {
	return func(ctx context.Context, in State) (State, error) {
		cl := client.DefaultClient
		if d := depsFrom(ctx); d != nil && d.client != nil {
			cl = d.client
		}
		body, _ := json.Marshal(map[string]string{"message": in.String()})
		req := cl.NewRequest(name, "Agent.Chat", &codecbytes.Frame{Data: body})
		var rsp codecbytes.Frame
		if err := cl.Call(ctx, req, &rsp); err != nil {
			return in, err
		}
		var out struct {
			Reply string `json:"reply"`
		}
		_ = json.Unmarshal(rsp.Data, &out)
		in.Data = []byte(out.Reply)
		return in, nil
	}
}

// LLM returns a StepFunc that runs one augmented-LLM turn: it renders the
// prompt template against the current state (.Data, .Stage), lets the
// model call the flow's services as tools, and stores the reply as the
// new Data.
func LLM(prompt string) StepFunc {
	return func(ctx context.Context, in State) (State, error) {
		d := depsFrom(ctx)
		if d == nil || d.model == nil {
			return in, fmt.Errorf("LLM step requires a flow model (set Provider/APIKey)")
		}
		text := prompt
		if tmpl, err := template.New("step").Parse(prompt); err == nil {
			var buf bytes.Buffer
			if tmpl.Execute(&buf, map[string]string{"Data": in.String(), "Stage": in.Stage}) == nil {
				text = buf.String()
			}
		}
		var tools []ai.Tool
		if d.tools != nil {
			tools, _ = d.tools.Discover()
		}
		resp, err := d.model.Generate(ctx, &ai.Request{Prompt: text, Tools: tools})
		if err != nil {
			return in, err
		}
		reply := resp.Answer
		if reply == "" {
			reply = resp.Reply
		}
		in.Data = []byte(reply)
		return in, nil
	}
}

// startRun begins a fresh run of the flow's steps with the given input.
func (f *Flow) startRun(ctx context.Context, data string) (Run, error) {
	run := Run{
		ID:      uuid.New().String(),
		Flow:    f.name,
		State:   State{Stage: f.opts.Steps[0].Name, Data: []byte(data)},
		Status:  "running",
		Started: time.Now(),
	}
	for _, s := range f.opts.Steps {
		run.Steps = append(run.Steps, StepRecord{Name: s.Name, Status: "pending"})
	}
	return f.runFrom(ctx, run)
}

// Resume continues a persisted run by id, picking up at the step it
// stopped on. Completed runs are a no-op.
func (f *Flow) Resume(ctx context.Context, runID string) error {
	if f.checkpoint == nil {
		return fmt.Errorf("flow %s has no checkpoint configured", f.name)
	}
	run, ok, err := f.checkpoint.Load(ctx, runID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("run %s not found", runID)
	}
	if run.Status == "done" {
		return nil
	}
	_, err = f.runFrom(ctx, run)
	return err
}

// Pending returns this flow's runs that have not completed — the ones a
// process restart should resume.
func (f *Flow) Pending(ctx context.Context) ([]Run, error) {
	if f.checkpoint == nil {
		return nil, nil
	}
	all, err := f.checkpoint.List(ctx)
	if err != nil {
		return nil, err
	}
	var out []Run
	for _, r := range all {
		if r.Flow == f.name && r.Status != "done" {
			out = append(out, r)
		}
	}
	return out, nil
}

// runFrom executes steps from the run's current Stage to the end,
// checkpointing before and after each step.
func (f *Flow) runFrom(ctx context.Context, run Run) (Run, error) {
	steps := f.opts.Steps
	ctx = withDeps(ctx, &runDeps{client: f.client, model: f.model, tools: f.toolSet})

	start := stepIndex(steps, run.State.Stage)
	if start < 0 {
		if run.State.Stage == "" {
			start = len(steps) // already finished
		} else {
			start = 0
		}
	}

	for i := start; i < len(steps); i++ {
		step := steps[i]
		run.State.Stage = step.Name
		run.Steps[i].Status = "in_progress"
		f.save(ctx, run)

		out, attempts, err := f.runStep(ctx, step, run.State)
		run.Steps[i].Attempts = attempts
		if err != nil {
			run.Steps[i].Status = "failed"
			run.Steps[i].Error = err.Error()
			run.Status = "failed"
			f.save(ctx, run)
			f.record(resultFromRun(f.opts.TriggerTopic, run))
			f.log.Logf(logger.ErrorLevel, "Flow %s run %s failed at step %q: %v", f.name, run.ID, step.Name, err)
			return run, err
		}

		run.State = out
		run.Steps[i].Status = "done"
		run.Steps[i].Result = truncate(out.String(), 200)
		if i+1 < len(steps) {
			run.State.Stage = steps[i+1].Name
		} else {
			run.State.Stage = ""
		}
		f.save(ctx, run)
	}

	run.Status = "done"
	f.save(ctx, run)
	if f.opts.DeleteOnSuccess && f.checkpoint != nil {
		_ = f.checkpoint.Delete(ctx, run.ID)
	}
	f.record(resultFromRun(f.opts.TriggerTopic, run))
	f.log.Logf(logger.InfoLevel, "Flow %s run %s completed (%d steps)", f.name, run.ID, len(steps))
	return run, nil
}

// runStep runs one step, retrying on error up to the resolved retry count.
func (f *Flow) runStep(ctx context.Context, step Step, in State) (State, int, error) {
	retries := f.opts.Retry
	if step.Retry > 0 {
		retries = step.Retry
	}
	var lastErr error
	for attempt := 1; attempt <= retries+1; attempt++ {
		out, err := step.Run(ctx, in)
		if err == nil {
			return out, attempt, nil
		}
		lastErr = err
	}
	return in, retries + 1, lastErr
}

func (f *Flow) save(ctx context.Context, run Run) {
	if f.checkpoint == nil {
		return
	}
	if err := f.checkpoint.Save(ctx, run); err != nil {
		f.log.Logf(logger.ErrorLevel, "Flow %s checkpoint save: %v", f.name, err)
	}
}

func stepIndex(steps []Step, name string) int {
	for i, s := range steps {
		if s.Name == name {
			return i
		}
	}
	return -1
}

func resultFromRun(trigger string, run Run) Result {
	r := Result{
		FlowName:  run.Flow,
		Trigger:   trigger,
		Timestamp: run.Started,
		Duration:  run.Updated.Sub(run.Started).Seconds(),
	}
	for _, s := range run.Steps {
		r.ToolCalls = append(r.ToolCalls, s.Name+":"+s.Status)
		if s.Error != "" {
			r.Error = s.Error
		}
	}
	if run.Status == "done" {
		r.Answer = run.State.String()
	}
	return r
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

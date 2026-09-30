// Package runstate holds shared execution records without depending on agents or flows.
package runstate

import (
	"context"
	"encoding/json"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
	"sort"
	"time"
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

// StepRecord is the recorded outcome of one step within a run.
type StepRecord struct {
	Name               string `json:"name"`
	Status             string `json:"status"` // pending | in_progress | done | failed
	Attempts           int    `json:"attempts"`
	Service            string `json:"service,omitempty"`
	Endpoint           string `json:"endpoint,omitempty"`
	Agent              string `json:"agent,omitempty"`
	ChildRunID         string `json:"child_run_id,omitempty"`
	Result             string `json:"result,omitempty"`
	Error              string `json:"error,omitempty"`
	ErrorKind          string `json:"error_kind,omitempty"`
	VerificationStatus string `json:"verification_status,omitempty"` // passed | failed
	VerificationNote   string `json:"verification_note,omitempty"`
}

// Run is the persisted record of one flow execution — what a Checkpoint
// saves and loads. It is retained for success and failure unless the flow
// opts into cleanup with DeleteOnSuccess.
type Run struct {
	ModelTurns   int                 `json:"model_turns,omitempty"`
	TurnAttempts int                 `json:"turn_attempts,omitempty"`
	PendingTurn  *model.Response     `json:"pending_turn,omitempty"`
	Continuation *model.Continuation `json:"continuation,omitempty"`
	ToolSteps    int                 `json:"tool_steps,omitempty"`
	CallCounts   map[string]int      `json:"call_counts,omitempty"`
	Spend        int64               `json:"spend,omitempty"`

	SchemaVersion int `json:"schema_version,omitempty"`

	ID         string       `json:"id"`
	ParentID   string       `json:"parent_id,omitempty"`
	Flow       string       `json:"flow"`
	OriginFlow string       `json:"origin_flow,omitempty"`
	OriginStep string       `json:"origin_step,omitempty"`
	Dispatch   string       `json:"dispatch,omitempty"`
	Trigger    string       `json:"trigger,omitempty"`
	State      State        `json:"state"`
	Steps      []StepRecord `json:"steps"`
	Status     string       `json:"status"` // running | waiting | done | failed
	Await      *AwaitState  `json:"await,omitempty"`
	Started    time.Time    `json:"started"`
	Updated    time.Time    `json:"updated"`
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
	store store.Store
}

// StoreCheckpoint returns a store-backed Checkpoint that keeps a flow's
// runs in their own store table — pass the flow name as scope, so one
// flow's runs never share a table with another's (or with service or
// agent state). A nil store uses store.DefaultStore.
func StoreCheckpoint(s store.Store, scope string) Checkpoint {
	if s == nil {
		s = store.DefaultStore
	}
	// Confine runs to the "flow" database, one table per flow name. The
	// scoped handle injects this per-operation without mutating s.
	return &storeCheckpoint{store: store.Scope(s, "flow", scope)}
}

func (c *storeCheckpoint) Save(ctx context.Context, run Run) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	run.SchemaVersion = 1
	run.Updated = time.Now()
	b, err := json.Marshal(run)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.store.Write(&store.Record{Key: run.ID, Value: b})
}

func (c *storeCheckpoint) Load(ctx context.Context, runID string) (Run, bool, error) {
	if err := ctx.Err(); err != nil {
		return Run{}, false, err
	}
	recs, err := c.store.Read(runID)
	if err == store.ErrNotFound || len(recs) == 0 {
		return Run{}, false, nil
	}
	if err != nil {
		return Run{}, false, err
	}
	var run Run
	if err := decode(recs[0].Value, &run); err != nil {
		return Run{}, false, err
	}
	return run, true, nil
}

func (c *storeCheckpoint) Delete(ctx context.Context, runID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.store.Delete(runID)
}

func (c *storeCheckpoint) List(ctx context.Context) ([]Run, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	keys, err := c.store.List()
	if err != nil {
		return nil, err
	}
	var runs []Run
	for _, id := range keys {
		run, ok, err := c.Load(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok {
			runs = append(runs, run)
		}
	}
	sort.SliceStable(runs, func(i, j int) bool {
		if runs[i].Started.Equal(runs[j].Started) {
			return runs[i].ID < runs[j].ID
		}
		return runs[i].Started.Before(runs[j].Started)
	})
	return runs, nil
}

// AwaitState records, on a suspended run, what it is waiting for.
type AwaitState struct {
	Step   string `json:"step"`
	Key    string `json:"key"`
	Prompt string `json:"prompt,omitempty"`
}

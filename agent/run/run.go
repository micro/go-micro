// Package run defines persisted agent execution records and checkpoint storage.
package run

import (
	"context"
	"encoding/json"
	"go-micro.dev/v6/store"
	"sort"
	"time"
)

// State holds the current stage and serialized input of a run.
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

// Run records one execution, including completed tools and pending input.
// The Flow field and JSON layout are retained for v6 checkpoint compatibility.
type Run struct {
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

// Checkpoint persists and restores runs so a run survives a crash
// and resumes where it stopped. The built-in StoreCheckpoint is
// store-backed; implement this interface to plug in another durable
// storage backend.
type Checkpoint interface {
	Save(ctx context.Context, run Run) error
	Load(ctx context.Context, runID string) (Run, bool, error)
	Delete(ctx context.Context, runID string) error
	List(ctx context.Context) ([]Run, error)
}

type storeCheckpoint struct {
	store store.Store
}

// StoreCheckpoint returns checkpoint storage scoped by agent or workflow name.
// A nil store uses store.DefaultStore. The v6 database name and record format
// are preserved so existing runs can still be resumed.
func StoreCheckpoint(s store.Store, scope string) Checkpoint {
	if s == nil {
		s = store.DefaultStore
	}
	// Keep the database name for compatibility with v6 checkpoints.
	return &storeCheckpoint{store: store.Scope(s, "flow", scope)}
}

func (c *storeCheckpoint) Save(ctx context.Context, run Run) error {
	if err := ctx.Err(); err != nil {
		return err
	}
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
	if err := json.Unmarshal(recs[0].Value, &run); err != nil {
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
		if run, ok, err := c.Load(ctx, id); err == nil && ok {
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

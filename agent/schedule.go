package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/store"
)

// Schedule is a persisted recurring request. One host owns scheduling for an
// agent/store namespace. Missed intervals coalesce into one run after restart;
// the next time is saved before dispatch, so interrupted work is not replayed.
type Schedule struct {
	ID       string        `json:"id"`
	Session  string        `json:"session"`
	Message  string        `json:"message"`
	Every    time.Duration `json:"every"`
	Next     time.Time     `json:"next"`
	LastTask string        `json:"last_task,omitempty"`
	Error    string        `json:"error,omitempty"`
}

type scheduleState struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

// ScheduleTask adds a recurring request to a running agent host. Definitions
// survive host restarts; execution requires Agent.Run to be active.
func ScheduleTask(ctx context.Context, a Agent, message string, every time.Duration) (*Schedule, error) {
	scheduler, ok := a.(interface {
		ScheduleTask(context.Context, string, time.Duration) (*Schedule, error)
	})
	if !ok {
		return nil, errors.New("agent: scheduling unsupported")
	}
	return scheduler.ScheduleTask(ctx, message, every)
}
func (a *agentImpl) ScheduleTask(ctx context.Context, message string, every time.Duration) (*Schedule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if message == "" || every < time.Minute {
		return nil, errors.New("agent: schedule needs a message and an interval of at least one minute")
	}
	id, _ := metadata.Get(ctx, "Micro-Session")
	if id == "" {
		id = a.opts.Session
	}
	if len(id) > 128 {
		return nil, errors.New("agent: session ID exceeds 128 bytes")
	}
	a.schedules.mu.Lock()
	defer a.schedules.mu.Unlock()
	entries, err := a.taskStore().List(store.ListPrefix("schedules/"))
	if err != nil {
		return nil, err
	}
	if len(entries) >= 100 {
		return nil, errors.New("agent: at most 100 schedules per host")
	}
	entry := &Schedule{ID: uuid.NewString(), Session: id, Message: message, Every: every, Next: time.Now().Add(every)}
	if err := a.taskStore().Write(store.NewRecord("schedules/"+entry.ID, entry)); err != nil {
		return nil, err
	}
	return entry, nil
}

func ListSchedules(s store.Store, name string) ([]Schedule, error) {
	st := store.Scope(s, "agent", name)
	keys, err := st.List(store.ListPrefix("schedules/"))
	if err != nil {
		return nil, err
	}
	var result []Schedule
	for _, key := range keys {
		records, err := st.Read(key)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			var entry Schedule
			if err := record.Decode(&entry); err != nil {
				return nil, err
			}
			result = append(result, entry)
		}
	}
	return result, nil
}

func (a *agentImpl) runSchedules(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := a.tickSchedules(ctx, time.Now()); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (a *agentImpl) tickSchedules(ctx context.Context, now time.Time) error {
	entries, err := ListSchedules(a.opts.Store, a.opts.Name)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Next.After(now) {
			continue
		}
		// Reserve the next interval before running. Remove cannot resurrect a
		// definition and a crash cannot cause automatic replay of this dispatch.
		a.schedules.mu.Lock()
		records, err := a.taskStore().Read("schedules/" + entry.ID)
		if err == nil && len(records) > 0 {
			err = records[0].Decode(&entry)
		}
		if errors.Is(err, store.ErrNotFound) || (err == nil && (len(records) == 0 || entry.Next.After(now))) {
			a.schedules.mu.Unlock()
			continue
		}
		if err == nil {
			entry.Next = now.Add(entry.Every)
			err = a.taskStore().Write(store.NewRecord("schedules/"+entry.ID, entry))
		}
		a.schedules.mu.Unlock()
		if err != nil {
			return err
		}
		f := flow.New(a.opts.Name+"-schedule-"+entry.ID,
			flow.WithCheckpoint(flow.StoreCheckpoint(a.opts.Store, a.opts.Name+"-schedules")),
			flow.Steps(flow.Step{Name: "agent", Run: func(ctx context.Context, in flow.State) (flow.State, error) {
				task, err := a.StartTask(WithSession(ctx, entry.Session), entry.Message)
				if err != nil {
					return in, err
				}
				entry.LastTask = task.ID
				result, err := a.waitTask(ctx, task.ID)
				if err != nil {
					return in, err
				}
				if result.Status != "done" {
					return in, fmt.Errorf("scheduled task %s: %s", task.ID, result.Error)
				}
				if result.Response != nil {
					in.Data = []byte(result.Response.Reply)
				}
				return in, nil
			}}))
		err = flow.Scheduled(f, entry.Message).Tick(ctx)
		entry.Error = ""
		if err != nil {
			entry.Error = err.Error()
		}
		a.schedules.mu.Lock()
		_, readErr := a.taskStore().Read("schedules/" + entry.ID)
		if readErr == nil {
			readErr = a.taskStore().Write(store.NewRecord("schedules/"+entry.ID, entry))
		}
		a.schedules.mu.Unlock()
		if readErr != nil && !errors.Is(readErr, store.ErrNotFound) {
			return readErr
		}
	}
	return nil
}

type scheduleHandler struct{ agent *agentImpl }

func scheduleResponse(s Schedule) *pb.ScheduleResponse {
	return &pb.ScheduleResponse{Id: s.ID, Session: s.Session, Message: s.Message, Every: s.Every.String(), Next: s.Next.Unix(), LastTask: s.LastTask, Error: s.Error}
}
func (h *scheduleHandler) Add(ctx context.Context, req *pb.AddScheduleRequest, rsp *pb.ScheduleResponse) error {
	every, err := time.ParseDuration(req.Every)
	if err != nil {
		return err
	}
	entry, err := h.agent.ScheduleTask(ctx, req.Message, every)
	if err != nil {
		return err
	}
	value := scheduleResponse(*entry)
	rsp.Id, rsp.Session, rsp.Message, rsp.Every, rsp.Next = value.Id, value.Session, value.Message, value.Every, value.Next
	return nil
}
func (h *scheduleHandler) List(ctx context.Context, _ *pb.ListSchedulesRequest, rsp *pb.ListSchedulesResponse) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	entries, err := ListSchedules(h.agent.opts.Store, h.agent.opts.Name)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		rsp.Schedules = append(rsp.Schedules, scheduleResponse(entry))
	}
	return nil
}

// RemoveSchedule deletes future dispatches; it does not cancel a running task.
func RemoveSchedule(ctx context.Context, a Agent, id string) error {
	scheduler, ok := a.(interface {
		RemoveSchedule(context.Context, string) error
	})
	if !ok {
		return errors.New("agent: scheduling unsupported")
	}
	return scheduler.RemoveSchedule(ctx, id)
}
func (a *agentImpl) RemoveSchedule(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.schedules.mu.Lock()
	defer a.schedules.mu.Unlock()
	return a.taskStore().Delete("schedules/" + id)
}
func (h *scheduleHandler) Remove(ctx context.Context, req *pb.RemoveScheduleRequest, _ *pb.RemoveScheduleResponse) error {
	return h.agent.RemoveSchedule(ctx, req.Id)
}

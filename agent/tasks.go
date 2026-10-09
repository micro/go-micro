package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/store"
)

// Task is work owned by a running agent host, independently of a chat client.
// Tasks survive client disconnects. A stopped or crashed host does not replay
// unfinished work; external tool actions may already have taken effect.
type Task struct {
	Address  string    `json:"address,omitempty"`
	ID       string    `json:"id"`
	Session  string    `json:"session"`
	Status   string    `json:"status"`
	Response *Response `json:"response,omitempty"`
	Error    string    `json:"error,omitempty"`
	Updated  time.Time `json:"updated"`
}

// TaskAgent is an optional background-work capability. Hosts must remain alive
// until their tasks finish and call Stop when shutting down.
type TaskAgent interface {
	StartTask(context.Context, string) (*Task, error)
	GetTask(context.Context, string) (*Task, error)
	StopTask(context.Context, string) error
}

func StartTask(ctx context.Context, a Agent, message string) (*Task, error) {
	host, ok := a.(TaskAgent)
	if !ok {
		return nil, errors.New("agent: background tasks unsupported")
	}
	return host.StartTask(ctx, message)
}
func GetTask(ctx context.Context, a Agent, id string) (*Task, error) {
	host, ok := a.(TaskAgent)
	if !ok {
		return nil, errors.New("agent: background tasks unsupported")
	}
	return host.GetTask(ctx, id)
}
func StopTask(ctx context.Context, a Agent, id string) error {
	host, ok := a.(TaskAgent)
	if !ok {
		return errors.New("agent: background tasks unsupported")
	}
	return host.StopTask(ctx, id)
}

type taskRun struct {
	cancel context.CancelFunc
	done   chan struct{}
	task   Task
}

type taskState struct {
	mu      sync.Mutex
	runs    map[string]*taskRun
	stopped bool
	address string
}

func (a *agentImpl) taskStore() store.Store { return store.Scope(a.opts.Store, "agent", a.opts.Name) }

func (a *agentImpl) StartTask(ctx context.Context, message string) (*Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if message == "" {
		return nil, errors.New("agent: task message is empty")
	}
	session, _ := metadata.Get(ctx, "Micro-Session")
	if session == "" {
		session = a.opts.Session
	}
	if len(session) > 128 {
		return nil, errors.New("agent: session ID exceeds 128 bytes")
	}
	a.tasks.mu.Lock()
	defer a.tasks.mu.Unlock()
	if a.tasks.stopped {
		return nil, errors.New("agent: host is stopping")
	}
	if len(a.tasks.runs) >= 16 {
		return nil, errors.New("agent: at most 16 background tasks may run at once")
	}
	if a.tasks.runs == nil {
		a.tasks.runs = map[string]*taskRun{}
	}
	task := Task{ID: uuid.NewString(), Session: session, Status: "running", Updated: time.Now(), Address: a.tasks.address}
	if err := a.taskStore().Write(store.NewRecord("tasks/"+task.ID, task)); err != nil {
		return nil, err
	}
	workCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	run := &taskRun{cancel: cancel, done: make(chan struct{}), task: task}
	a.tasks.runs[task.ID] = run
	go a.runTask(workCtx, run, message)
	return &task, nil
}

func (a *agentImpl) runTask(ctx context.Context, run *taskRun, message string) {
	response, err := func() (*Response, error) {
		conversation, release, err := a.rpcSession(ctx)
		if err != nil {
			return nil, err
		}
		defer release()
		conversation.mu.Lock()
		defer conversation.mu.Unlock()
		conversation.setup()
		return conversation.askLocked(ctx, run.task.ID, message, conversation.parentRunID, nil, true)
	}()
	a.tasks.mu.Lock()
	defer a.tasks.mu.Unlock()
	defer close(run.done)
	defer run.cancel()
	run.task.Status = "done"
	run.task.Response = response
	run.task.Updated = time.Now()
	if err != nil {
		run.task.Status = "error"
		run.task.Error = err.Error()
	}
	if errors.Is(err, context.Canceled) {
		run.task.Status = "canceled"
	}
	if err := a.taskStore().Write(store.NewRecord("tasks/"+run.task.ID, run.task)); err != nil {
		run.task.Status = "error"
		run.task.Error = fmt.Sprintf("save task result: %v", err)
		// Retain failed writes in memory so Get reports the error instead of success.
		return
	}
	delete(a.tasks.runs, run.task.ID)
}

func (a *agentImpl) GetTask(ctx context.Context, id string) (*Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	a.tasks.mu.Lock()
	defer a.tasks.mu.Unlock()
	if run := a.tasks.runs[id]; run != nil {
		task := run.task
		return &task, nil
	}
	records, err := a.taskStore().Read("tasks/" + id)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, store.ErrNotFound
	}
	var task Task
	if err := records[0].Decode(&task); err != nil {
		return nil, err
	}
	if task.Status == "running" {
		task.Status = "interrupted"
		task.Error = "The owning host stopped. This task was not replayed."
		if task.Address != "" && task.Address != a.tasks.address {
			task.Status = "unknown"
			task.Error = "Connect to the recorded owner address to inspect this task. It has not been replayed."
		}
	}
	return &task, nil
}

func (a *agentImpl) StopTask(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.tasks.mu.Lock()
	defer a.tasks.mu.Unlock()
	if run := a.tasks.runs[id]; run != nil {
		run.cancel()
		return nil
	}
	return errors.New("agent: task is not running on this host")
}

func (a *agentImpl) stopTasks() {
	a.tasks.mu.Lock()
	a.tasks.stopped = true
	runs := make([]*taskRun, 0, len(a.tasks.runs))
	for _, run := range a.tasks.runs {
		run.cancel()
		runs = append(runs, run)
	}
	a.tasks.mu.Unlock()
	for _, run := range runs {
		<-run.done
	}
}

type taskHandler struct{ agent *agentImpl }

func (h *taskHandler) response(task *Task, rsp *pb.TaskResponse) {
	rsp.Id, rsp.Session, rsp.Status, rsp.Error = task.ID, task.Session, task.Status, task.Error
	if task.Response != nil {
		rsp.Reply = task.Response.Reply
	}
	if task.Address != "" {
		rsp.Address = task.Address
	}
	if h.agent.server != nil && (rsp.Address == "" || (task.Status != "running" && task.Status != "unknown")) {
		rsp.Address = h.agent.server.Options().Address
	}
}
func (h *taskHandler) Start(ctx context.Context, req *pb.StartTaskRequest, rsp *pb.TaskResponse) error {
	task, err := h.agent.StartTask(ctx, req.Message)
	if err != nil {
		return err
	}
	h.response(task, rsp)
	return nil
}
func (h *taskHandler) Get(ctx context.Context, req *pb.TaskRequest, rsp *pb.TaskResponse) error {
	task, err := h.agent.GetTask(ctx, req.Id)
	if err != nil {
		return err
	}
	h.response(task, rsp)
	return nil
}
func (h *taskHandler) Stop(ctx context.Context, req *pb.TaskRequest, rsp *pb.TaskResponse) error {
	if err := h.agent.StopTask(ctx, req.Id); err != nil {
		return err
	}
	return h.Get(ctx, req, rsp)
}
func (h *taskHandler) List(ctx context.Context, _ *pb.ListTasksRequest, rsp *pb.ListTasksResponse) error {
	keys, err := h.agent.taskStore().List(store.ListPrefix("tasks/"))
	if err != nil {
		return err
	}
	for _, key := range keys {
		task, err := h.agent.GetTask(ctx, key[len("tasks/"):])
		if err != nil {
			return err
		}
		item := &pb.TaskResponse{}
		h.response(task, item)
		rsp.Tasks = append(rsp.Tasks, item)
	}
	return nil
}

func (a *agentImpl) waitTask(ctx context.Context, id string) (*Task, error) {
	a.tasks.mu.Lock()
	run := a.tasks.runs[id]
	a.tasks.mu.Unlock()
	if run != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-run.done:
		}
	}
	return a.GetTask(ctx, id)
}

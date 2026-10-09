package agent

import (
	"context"
	"testing"
	"time"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

func TestTaskSurvivesCallerAndRecordsSession(t *testing.T) {
	started, finish := make(chan struct{}), make(chan struct{})
	fakeGen = func(ctx context.Context, _ model.Options, _ *model.Request) (*model.Response, error) {
		close(started)
		select {
		case <-finish:
			return &model.Response{Reply: "finished"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	defer func() { fakeGen = nil }()
	st := store.NewMemoryStore()
	a := newTestAgent(Name("worker"), WithStore(st))
	defer a.Stop()
	ctx, cancel := context.WithCancel(WithSession(context.Background(), "conversation"))
	task, err := a.StartTask(ctx, "work")
	if err != nil {
		t.Fatal(err)
	}
	<-started
	cancel()
	running, err := a.GetTask(context.Background(), task.ID)
	if err != nil || running.Status != "running" {
		t.Fatalf("task=%+v err=%v", running, err)
	}
	close(finish)
	completed := waitTask(t, a, task.ID)
	if completed.Status != "done" || completed.Response.Reply != "finished" || completed.Response.RunID != task.ID {
		t.Fatalf("task=%+v", completed)
	}
	history, err := LoadHistory(st, "worker", "conversation")
	if err != nil || len(history) != 2 {
		t.Fatalf("history=%+v err=%v", history, err)
	}
	sessions, err := ListSessions(st, "worker")
	if err != nil || len(sessions) != 1 || sessions[0].ID != "conversation" {
		t.Fatalf("sessions=%+v err=%v", sessions, err)
	}
	handler := sessionHandler{agent: a}
	var response pb.HistoryResponse
	if err := handler.History(context.Background(), &pb.HistoryRequest{Session: "conversation"}, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Messages) != 2 || response.Messages[1].Content != "finished" {
		t.Fatalf("response=%+v", response.Messages)
	}
}

func TestTaskStopsWithHost(t *testing.T) {
	started := make(chan struct{})
	fakeGen = func(ctx context.Context, _ model.Options, _ *model.Request) (*model.Response, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer func() { fakeGen = nil }()
	a := newTestAgent(Name("stop-worker"))
	task, err := a.StartTask(context.Background(), "work")
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}
	result, err := a.GetTask(context.Background(), task.ID)
	if err != nil || result.Status != "canceled" {
		t.Fatalf("task=%+v err=%v", result, err)
	}
	if _, err := a.StartTask(context.Background(), "again"); err == nil {
		t.Fatal("accepted task after shutdown")
	}
}

func waitTask(t *testing.T, a *agentImpl, id string) *Task {
	t.Helper()
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		result, err := a.GetTask(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != "running" {
			return result
		}
		select {
		case <-deadline:
			t.Fatal("task did not finish")
		case <-ticker.C:
		}
	}
}

func TestTaskRPCAndHistoryReconnect(t *testing.T) {
	fakeGen = func(context.Context, model.Options, *model.Request) (*model.Response, error) {
		return &model.Response{Reply: "rpc result"}, nil
	}
	defer func() { fakeGen = nil }()
	reg := registry.NewMemoryRegistry()
	a := newTestAgent(Name("rpc-worker"), Address("127.0.0.1:0"), WithRegistry(reg), WithBroker(broker.NewMemoryBroker()))
	if _, err := a.startServer(); err != nil {
		t.Fatal(err)
	}
	defer a.Stop()
	cl := client.NewClient(client.Registry(reg))
	api := pb.NewAgentTasksService(a.Name(), cl)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	task, err := api.Start(WithSession(ctx, "remote"), &pb.StartTaskRequest{Message: "work"}, client.WithRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	if task.Address == "" || task.Address == "127.0.0.1:0" {
		t.Fatalf("invalid owner address %q", task.Address)
	}
	waitTask(t, a, task.Id)
	// A fresh client attaches using the recorded owner address.
	api = pb.NewAgentTasksService(a.Name(), client.NewClient())
	result, err := api.Get(ctx, &pb.TaskRequest{Id: task.Id}, client.WithAddress(task.Address))
	if err != nil || result.Reply != "rpc result" {
		t.Fatalf("result=%v err=%v", result, err)
	}
	history, err := pb.NewAgentSessionsService(a.Name(), cl).History(ctx, &pb.HistoryRequest{Session: "remote"})
	if err != nil || len(history.Messages) != 2 {
		t.Fatalf("history=%v err=%v", history, err)
	}
}

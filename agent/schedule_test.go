package agent

import (
	"context"
	"testing"
	"time"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

func TestScheduleSurvivesRestartAndUsesTask(t *testing.T) {
	calls := 0
	fakeGen = func(context.Context, model.Options, *model.Request) (*model.Response, error) {
		calls++
		return &model.Response{Reply: "scheduled result"}, nil
	}
	defer func() { fakeGen = nil }()
	st := store.NewMemoryStore()
	first := newTestAgent(Name("scheduled"), WithStore(st))
	entry, err := ScheduleTask(WithSession(context.Background(), "daily"), first, "check the project", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	cp := StoreCheckpoint(st, "scheduled")
	restarted := newTestAgent(Name("scheduled"), WithStore(st), WithCheckpoint(cp))
	defer restarted.Stop()
	now := entry.Next.Add(time.Second)
	if err := restarted.tickSchedules(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if err := restarted.tickSchedules(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	entries, err := ListSchedules(st, "scheduled")
	if err != nil || len(entries) != 1 || entries[0].LastTask == "" || !entries[0].Next.After(now) {
		t.Fatalf("entries=%+v err=%v", entries, err)
	}
	task, err := restarted.GetTask(context.Background(), entries[0].LastTask)
	if err != nil || task.Status != "done" || task.Response.RunID != task.ID {
		t.Fatalf("task=%+v err=%v", task, err)
	}
	run, found, err := cp.Load(context.Background(), task.ID)
	if err != nil || !found || run.Status != "done" || run.Dispatch != "schedule" || run.Trigger != entry.ID || run.ParentID != "" {
		t.Fatalf("scheduled run=%+v found=%v err=%v", run, found, err)
	}
	wrapperRuns, err := StoreCheckpoint(st, "scheduled-schedules").List(context.Background())
	if err != nil || len(wrapperRuns) != 0 {
		t.Fatalf("unexpected workflow runs=%+v err=%v", wrapperRuns, err)
	}
	history, err := LoadHistory(st, "scheduled", "daily")
	if err != nil || len(history) != 2 || history[1].Content != "scheduled result" {
		t.Fatalf("history=%+v err=%v", history, err)
	}
	if err := RemoveSchedule(context.Background(), restarted, entry.ID); err != nil {
		t.Fatal(err)
	}
	entries, err = ListSchedules(st, "scheduled")
	if err != nil || len(entries) != 0 {
		t.Fatalf("entries=%+v err=%v", entries, err)
	}
}

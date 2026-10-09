package agent

import (
	"context"
	"testing"
	"time"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

func TestScheduleSurvivesRestartAndUsesFlow(t *testing.T) {
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
	restarted := newTestAgent(Name("scheduled"), WithStore(st))
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
	if err != nil || task.Status != "done" || task.Response.ParentID == "" {
		t.Fatalf("task=%+v err=%v", task, err)
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

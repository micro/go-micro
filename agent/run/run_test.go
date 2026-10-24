package run_test

import (
	"context"
	"testing"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/store"
)

func TestExistingCheckpointRemainsReadable(t *testing.T) {
	st := store.NewMemoryStore()
	legacy := store.Scope(st, "flow", "assistant")
	if err := legacy.Write(&store.Record{Key: "saved", Value: []byte(`{"id":"saved","flow":"assistant","state":{"stage":"ask","data":"aGk="},"steps":[{"name":"ask","status":"done","result":"hello"}],"status":"done"}`)}); err != nil {
		t.Fatal(err)
	}
	cp := agent.StoreCheckpoint(st, "assistant")
	record, found, err := cp.Load(context.Background(), "saved")
	if err != nil || !found || record.State.String() != "hi" || record.Steps[0].Result != "hello" {
		t.Fatalf("run=%+v found=%v err=%v", record, found, err)
	}
	var old flow.Checkpoint = cp
	record.Status = "waiting"
	if err := old.Save(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	restored, found, err := flow.StoreCheckpoint(st, "assistant").Load(context.Background(), "saved")
	if err != nil || !found || restored.Status != "waiting" {
		t.Fatalf("run=%+v found=%v err=%v", restored, found, err)
	}
}

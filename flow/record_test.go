package flow

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-micro.dev/v6/store"
)

func TestLoadRunRecordReturnsVersionedFlowExecution(t *testing.T) {
	ctx := context.Background()
	cp := StoreCheckpoint(store.NewMemoryStore(), "checkout")
	want := Run{
		ID: "flow-run-1", Flow: "checkout", Dispatch: "schedule", Trigger: "nightly", Status: "done",
		Started: time.Unix(100, 0).UTC(),
		Steps: []StepRecord{{Name: "charge", Status: "done", Attempts: 1, Service: "payments", Endpoint: "Payments.Charge"},
			{Name: "notify", Status: "done", Attempts: 1, Agent: "comms", ChildRunID: "agent-run-1"}},
	}
	if err := cp.Save(ctx, want); err != nil {
		t.Fatal(err)
	}
	record, err := LoadRunRecord(ctx, cp, "checkout", want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.SchemaVersion != RunRecordSchemaVersion || record.Run.ID != want.ID ||
		record.Run.Steps[1].ChildRunID != "agent-run-1" {
		t.Fatalf("record = %#v", record)
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	var decoded RunRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SchemaVersion != RunRecordSchemaVersion || decoded.Run.Steps[0].Service != "payments" {
		t.Fatalf("decoded record = %#v", decoded)
	}
}

func TestLoadRunRecordMissingCarriesRequestedIdentity(t *testing.T) {
	record, err := LoadRunRecord(context.Background(), StoreCheckpoint(store.NewMemoryStore(), "checkout"), "checkout", "missing")
	if err != nil {
		t.Fatal(err)
	}
	if record.SchemaVersion != RunRecordSchemaVersion || record.Run.Flow != "checkout" || record.Run.ID != "missing" || record.Run.Status != "" {
		t.Fatalf("missing record = %#v", record)
	}
}

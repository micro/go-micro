package flow

import "context"

// RunRecordSchemaVersion is the current JSON schema version for RunRecord.
const RunRecordSchemaVersion = 1

// RunRecord is the versioned, durable representation of one flow execution.
// Consumers should use SchemaVersion when decoding records across releases.
type RunRecord struct {
	SchemaVersion int `json:"schema_version"`
	Run           Run `json:"run"`
}

// LoadRunRecord loads one versioned flow run from checkpoint storage. A
// missing run returns an empty record carrying the requested flow and run id.
func LoadRunRecord(ctx context.Context, checkpoint Checkpoint, flowName, runID string) (RunRecord, error) {
	if checkpoint == nil {
		checkpoint = StoreCheckpoint(nil, flowName)
	}
	run, ok, err := checkpoint.Load(ctx, runID)
	if err != nil {
		return RunRecord{}, err
	}
	if !ok {
		run = Run{ID: runID, Flow: flowName}
	}
	return RunRecord{SchemaVersion: RunRecordSchemaVersion, Run: run}, nil
}

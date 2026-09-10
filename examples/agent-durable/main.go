// Package main demonstrates durable agent runs: a checkpointed agent can
// resume after a crash without re-executing completed tool calls.
package main

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	micro "go-micro.dev/v6"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

func main() {
	ctx := context.Background()
	checkpoint := micro.StoreCheckpoint(store.NewMemoryStore(), "durable-agent-demo")
	m := &demoModel{failFirst: true}
	model.Register("durable-demo", func(opts ...model.Option) model.Model {
		_ = m.Init(opts...)
		return m
	})
	var reservations atomic.Int32

	ag := micro.NewAgent("durable-agent-demo",
		micro.AgentWithCheckpoint(checkpoint),
		micro.AgentProvider("durable-demo"),
		micro.AgentTool("inventory.reserve", "reserve inventory exactly once", map[string]any{
			"sku": map[string]any{"type": "string"},
		}, func(ctx context.Context, input map[string]any) (string, error) {
			count := reservations.Add(1)
			return fmt.Sprintf("reserved %s (execution %d)", input["sku"], count), nil
		}),
	)

	_, err := ag.Ask(ctx, "reserve sku-123 and confirm")
	fmt.Println("initial run:", err)

	pending, err := micro.AgentPending(ctx, ag)
	if err != nil {
		panic(err)
	}
	if len(pending) == 0 {
		panic("expected a checkpointed run to resume")
	}

	resp, err := micro.AgentResume(ctx, ag, pending[0].ID)
	if err != nil {
		panic(err)
	}
	fmt.Println("resumed reply:", resp.Reply)
	fmt.Println("tool executions:", reservations.Load())
}

type demoModel struct {
	failFirst bool
	opts      model.Options
}

func (m *demoModel) Init(opts ...model.Option) error {
	m.opts = model.NewOptions(opts...)
	return nil
}
func (m *demoModel) Options() model.Options { return m.opts }
func (m *demoModel) String() string         { return "durable-demo" }

func (m *demoModel) Generate(ctx context.Context, req *model.Request, opts ...model.GenerateOption) (*model.Response, error) {
	if m.opts.ToolHandler != nil {
		res := m.opts.ToolHandler(ctx, model.ToolCall{
			ID:    "reserve-1",
			Name:  "inventory.reserve",
			Input: map[string]any{"sku": "sku-123"},
		})
		if res.Content == "" {
			return nil, errors.New("reservation tool returned no content")
		}
	}
	if m.failFirst {
		m.failFirst = false
		return nil, errors.New("simulated process interruption after checkpointed tool call")
	}
	return &model.Response{Reply: "sku-123 is reserved; no duplicate reservation was made"}, nil
}

func (m *demoModel) Stream(context.Context, *model.Request, ...model.GenerateOption) (model.Stream, error) {
	return nil, model.ErrStreamingUnsupported
}

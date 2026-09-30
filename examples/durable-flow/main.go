// durable-flow demonstrates services, an agent, and an approval flow across restarts.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
	"go-micro.dev/v6/service"
	bolt "go.etcd.io/bbolt"
)

type Request struct {
	Text string `json:"text"`
}
type Response struct {
	Text string `json:"text"`
}
type Ledger struct{ db *bolt.DB }

func (l *Ledger) Lookup(_ context.Context, _ *Request, out *Response) error {
	out.Text = "reviewed"
	return nil
}
func (l *Ledger) Prepare(ctx context.Context, in *Request, out *Response) error {
	return l.record(ctx, in, out)
}
func (l *Ledger) Commit(ctx context.Context, in *Request, out *Response) error {
	return l.record(ctx, in, out)
}
func (l *Ledger) record(ctx context.Context, in *Request, out *Response) error {
	key, ok := metadata.Get(ctx, "micro-idempotency-key")
	if !ok || key == "" {
		return fmt.Errorf("missing idempotency key")
	}
	return l.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("effects"))
		if err != nil {
			return err
		}
		if value := b.Get([]byte(key)); value != nil {
			return json.Unmarshal(value, out)
		}
		out.Text = in.Text
		data, err := json.Marshal(out)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), data)
	})
}

type mockModel struct{ opts model.Options }

func (m *mockModel) Init(opts ...model.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *mockModel) Options() model.Options { return m.opts }
func (m *mockModel) String() string         { return "durable-demo" }
func (m *mockModel) Stream(context.Context, *model.Request, ...model.GenerateOption) (model.Stream, error) {
	return nil, model.ErrStreamingUnsupported
}
func (m *mockModel) Generate(ctx context.Context, r *model.Request, o ...model.GenerateOption) (*model.Response, error) {
	return m.Turn(ctx, r, o...)
}
func (m *mockModel) Turn(_ context.Context, r *model.Request, _ ...model.GenerateOption) (*model.Response, error) {
	if r.Continuation != nil {
		return &model.Response{Reply: `{"text":"review approved"}`, StopReason: "stop"}, nil
	}
	for _, tool := range r.Tools {
		if tool.OriginalName == "work-ledger.Ledger.Lookup" {
			return &model.Response{ToolCalls: []model.ToolCall{{ID: "lookup", Name: tool.Name, Input: map[string]any{}}}, Continuation: &model.Continuation{Provider: m.String()}}, nil
		}
	}
	return nil, fmt.Errorf("ledger service tool was not discovered")
}

func run(dir, id, action string) error { return runConfigured(dir, id, action, "durable-demo", "") }
func runConfigured(dir, id, action, provider, modelName string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	cp, err := flow.OpenCheckpoint(filepath.Join(dir, "runs.db"))
	if err != nil {
		return err
	}
	defer cp.Close()
	if action == "status" {
		r, ok, err := cp.Load(context.Background(), id)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("run not found")
		}
		data, err := json.MarshalIndent(r, "", "  ")
		if err == nil {
			fmt.Println(string(data))
		}
		return err
	}
	db, err := bolt.Open(filepath.Join(dir, "effects.db"), 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	reg := registry.NewMemoryRegistry()
	br := broker.NewMemoryBroker()
	cl := client.NewClient(client.Registry(reg), client.Selector(selector.NewSelector(selector.Registry(reg))), client.Broker(br))
	svc := service.New(service.Name("work-ledger"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl), service.Broker(br), service.HandleSignal(false))
	if err := svc.Handle(&Ledger{db: db}); err != nil {
		return err
	}
	if err := svc.Start(); err != nil {
		return err
	}
	defer svc.Stop()
	model.Register("durable-demo", func(opts ...model.Option) model.Model { return &mockModel{opts: model.NewOptions(opts...)} })
	f := flow.New("review", flow.Provider(provider), flow.Model(modelName), flow.APIKey(os.Getenv("MICRO_AI_API_KEY")), flow.BaseURL(os.Getenv("MICRO_AI_BASE_URL")), flow.StrictRecovery(), flow.WithCheckpoint(cp), flow.AgentOptions(agent.Services("work-ledger"), agent.MaxSteps(4)), flow.Steps(
		flow.Step{Name: "prepare", Idempotent: true, Run: flow.Call("work-ledger", "Ledger.Prepare")},
		flow.Step{Name: "review", Idempotent: true, Run: flow.LLM("Review {{.Data}} using the ledger")},
		flow.AwaitStep("approval", "approval", "Approve the reviewed work?"),
		flow.Step{Name: "commit", Idempotent: true, Run: flow.Call("work-ledger", "Ledger.Commit")},
	))
	if err := f.Register(reg, br, cl); err != nil {
		return err
	}
	defer f.Stop()
	switch action {
	case "start":
		_, err = f.Start(context.Background(), id, `{"text":"review this work"}`)
	case "approve":
		err = f.ResumeWith(context.Background(), id, `{"text":"approved"}`)
	case "resume":
		err = f.Resume(context.Background(), id)
	default:
		return fmt.Errorf("action must be start, approve, resume, or status")
	}
	if err != nil {
		return err
	}
	r, _, err := cp.Load(context.Background(), id)
	fmt.Printf("run=%s status=%s stage=%s\n", r.ID, r.Status, r.State.Stage)
	return err
}
func main() {
	dir := flag.String("dir", ".durable-flow", "Journal directory")
	id := flag.String("id", "review-1", "Stable run ID")
	action := flag.String("action", "start", "start, approve, resume, status")
	provider := flag.String("provider", "durable-demo", "Model provider; default requires no network or key")
	modelName := flag.String("model", "", "Provider model name")
	flag.Parse()
	if err := runConfigured(*dir, *id, *action, *provider, *modelName); err != nil {
		log.Fatal(err)
	}
}

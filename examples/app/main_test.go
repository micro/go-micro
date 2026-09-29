package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
	"go-micro.dev/v6/service"
	"go-micro.dev/v6/store"
)

type notesModel struct{ opts model.Options }

func (m *notesModel) Init(opts ...model.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *notesModel) Options() model.Options { return m.opts }
func (m *notesModel) String() string         { return "app-test" }
func (m *notesModel) Stream(context.Context, *model.Request, ...model.GenerateOption) (model.Stream, error) {
	return nil, model.ErrStreamingUnsupported
}
func (m *notesModel) Generate(ctx context.Context, req *model.Request, _ ...model.GenerateOption) (*model.Response, error) {
	for _, tool := range req.Tools {
		if strings.HasSuffix(tool.OriginalName, "Notes.Add") {
			result := m.opts.ToolHandler(ctx, model.ToolCall{ID: "add", Name: tool.Name, Input: map[string]any{"text": req.Prompt}})
			return &model.Response{Answer: result.Content}, nil
		}
	}
	return nil, fmt.Errorf("notes tool not discovered")
}

func TestUIAndAgentShareService(t *testing.T) {
	r := registry.NewMemoryRegistry()
	br := broker.NewMemoryBroker()
	cl := client.NewClient(client.Registry(r), client.Selector(selector.NewSelector(selector.Registry(r))), client.Broker(br))
	svc := service.New(service.Name("app-notes"), service.Address("127.0.0.1:0"), service.Registry(r), service.Client(cl), service.Broker(br), service.HandleSignal(false))
	if err := svc.Handle(new(Notes)); err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	defer svc.Stop()
	a, handler, err := newApplication(cl)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CheckDependencies(r); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/notes", strings.NewReader(`{"text":"from the UI"}`)))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	model.Register("app-test", func(opts ...model.Option) model.Model { m := &notesModel{}; _ = m.Init(opts...); return m })
	ag := agent.New(agent.Name("notes-assistant"), agent.Provider("app-test"), agent.Services("app-notes"), agent.WithRegistry(r), agent.WithClient(cl), agent.WithStore(store.NewMemoryStore()))
	if _, err := ag.Ask(context.Background(), "from the agent"); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/notes", nil))
	var rsp NotesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &rsp); err != nil {
		t.Fatal(err)
	}
	if len(rsp.Notes) != 2 || rsp.Notes[0] != "from the UI" || rsp.Notes[1] != "from the agent" {
		t.Fatalf("notes=%v", rsp.Notes)
	}
}

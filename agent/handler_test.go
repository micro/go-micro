package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/server"
	"go-micro.dev/v6/service"
	"go-micro.dev/v6/store"
)

type Weather struct{}
type WeatherRequest struct {
	Name string `json:"name"`
}
type WeatherResponse struct {
	Msg string `json:"msg"`
}

func (*Weather) Call(_ context.Context, req *WeatherRequest, rsp *WeatherResponse) error {
	rsp.Msg = "Hello " + req.Name
	return nil
}

func TestServiceHostsAgent(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	svc := service.New(service.Name("weather"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Store(store.NewMemoryStore()), service.Broker(broker.NewMemoryBroker()))
	if err := svc.Server().Init(server.Metadata(map[string]string{"type": "service", "custom": "kept"})); err != nil {
		t.Fatal(err)
	}
	a := New(Provider("fake")).(*agentImpl)
	if err := svc.Handle(&Weather{}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Handle(a); err != nil {
		t.Fatal(err)
	}
	if err := svc.Handle(a); err == nil {
		t.Fatal("attached twice")
	}
	if _, err := a.startServer(); err == nil {
		t.Fatal("attached agent started a separate server")
	}
	fakeGen = func(ctx context.Context, opts model.Options, req *model.Request) (*model.Response, error) {
		var name string
		for _, tool := range req.Tools {
			if strings.Contains(tool.OriginalName, ".Agent") {
				t.Errorf("control endpoint offered as tool: %s", tool.OriginalName)
			}
			if tool.OriginalName == "weather.Weather.Call" {
				name = tool.Name
			}
		}
		if name == "" {
			t.Error("own business handler missing")
		}
		result := opts.ToolHandler(ctx, model.ToolCall{Name: name, Input: map[string]any{"name": "Alice"}})
		return &model.Response{Reply: result.Content}, nil
	}
	defer func() { fakeGen = nil }()
	if err := svc.Start(); err != nil {
		t.Fatal(err)
	}
	defer svc.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cl := client.NewClient(client.Registry(reg))
	var rsp WeatherResponse
	if err := cl.Call(ctx, cl.NewRequest(svc.Name(), "Weather.Call", &WeatherRequest{Name: "Bob"}), &rsp); err != nil || rsp.Msg != "Hello Bob" {
		t.Fatalf("RPC: %+v %v", rsp, err)
	}
	reply, err := pb.NewAgentService(svc.Name(), cl).Chat(ctx, &pb.ChatRequest{Message: "Greet Alice"})
	if err != nil || !strings.Contains(reply.GetReply(), "Hello Alice") {
		t.Fatalf("chat: %+v %v", reply, err)
	}
	records, err := reg.GetService(svc.Name())
	if err != nil || len(records) != 1 || len(records[0].Nodes) != 1 {
		t.Fatalf("one shared registration: %+v %v", records, err)
	}
	md := records[0].Nodes[0].Metadata
	if md["agent"] != "v1" || md["type"] != "service" || md["custom"] != "kept" {
		t.Fatalf("metadata: %v", md)
	}
	if a.tasks.address != svc.Server().Options().Address {
		t.Fatal("task host uses a different address")
	}
	if err := svc.Stop(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.StartTask(ctx, "after stop"); err == nil {
		t.Fatal("task accepted after service stop")
	}
	if records, err := reg.GetService(svc.Name()); err == nil && len(records) > 0 {
		t.Fatal("service still registered")
	}
}

func TestAgentControlEndpoints(t *testing.T) {
	for _, handler := range []string{"Agent", "AgentInteraction", "AgentSessions", "AgentTasks", "AgentSchedules"} {
		if !agentEndpoint("org.weather." + handler + ".Call") {
			t.Fatal(handler)
		}
	}
	for _, name := range []string{"org.weather.Weather.Call", "weather.AgentTools.Call", "invalid"} {
		if agentEndpoint(name) {
			t.Fatal(name)
		}
	}
}

func TestHostedAgentStopsTasksOnStartFailure(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	svc := service.New(service.Name("worker"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Store(store.NewMemoryStore()), service.Broker(broker.NewMemoryBroker()))
	a := New(Provider("fake")).(*agentImpl)
	if err := svc.Handle(a); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	fakeGen = func(ctx context.Context, _ model.Options, _ *model.Request) (*model.Response, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer func() { fakeGen = nil }()
	var task *Task
	// Append after the agent's startup hook without parsing the test process flags.
	opts := svc.Options()
	opts.AfterStart = append(opts.AfterStart, func() error {
		var err error
		task, err = a.StartTask(context.Background(), "work")
		if err != nil {
			return err
		}
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			t.Error("task did not start")
		}
		return errors.New("later startup hook failed")
	})
	// The test uses the same registrar mechanism as an application component.
	if err := svc.Handle(startFailure{hooks: opts.AfterStart}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(); err == nil || !strings.Contains(err.Error(), "later startup") {
		t.Fatalf("start: %v", err)
	}
	if task == nil {
		t.Fatal("no task")
	}
	result, err := a.GetTask(context.Background(), task.ID)
	if err != nil || result.Status != "canceled" {
		t.Fatalf("task: %+v %v", result, err)
	}
	if records, err := reg.GetService(svc.Name()); err == nil && len(records) > 0 {
		t.Fatal("failed host still registered")
	}
}

type startFailure struct{ hooks []func() error }

func (h startFailure) Register(o *service.Options) error { o.AfterStart = h.hooks; return nil }

// A hosted scheduler must return its error through service.Run, just as a
// standalone agent returns it through agent.Run.
func TestHostedAgentReturnsScheduleError(t *testing.T) {
	want := errors.New("store unavailable")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	svc := service.New(service.Name("broken"), service.Context(ctx), service.HandleSignal(false), service.Address("127.0.0.1:0"), service.Registry(registry.NewMemoryRegistry()), service.Store(failedScheduleStore{Store: store.NewMemoryStore(), err: want}), service.Broker(broker.NewMemoryBroker()))
	if err := svc.Handle(New(Provider("fake"))); err != nil {
		t.Fatal(err)
	}
	if err := svc.Run(); !errors.Is(err, want) {
		t.Fatalf("Run error = %v", err)
	}
}

type failedScheduleStore struct {
	store.Store
	err error
}

func (s failedScheduleStore) List(...store.ListOption) ([]string, error) { return nil, s.err }

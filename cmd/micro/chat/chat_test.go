package chat

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
	"os/exec"
	"testing"

	"go-micro.dev/v6/client"
	"go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

type harnessModel struct {
	opts     model.Options
	generate func(context.Context, *model.Request, model.Options) (*model.Response, error)
}

func (m *harnessModel) Init(opts ...model.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *harnessModel) Options() model.Options { return m.opts }
func (m *harnessModel) String() string         { return "chat-test" }
func (m *harnessModel) Generate(ctx context.Context, r *model.Request, _ ...model.GenerateOption) (*model.Response, error) {
	return m.generate(ctx, r, m.opts)
}
func (m *harnessModel) Stream(context.Context, *model.Request, ...model.GenerateOption) (model.Stream, error) {
	return nil, model.ErrStreamingUnsupported
}

func testSession(t *testing.T, fn func(context.Context, *model.Request, model.Options) (*model.Response, error)) *session {
	t.Helper()
	name := t.Name()
	model.Register(name, func(opts ...model.Option) model.Model {
		return &harnessModel{opts: model.NewOptions(opts...), generate: fn}
	})
	return &session{provider: name, modelName: "chosen-model", baseURL: "http://model.invalid", reg: registry.NewMemoryRegistry(), cl: client.DefaultClient}
}

func TestHarnessConversationAndReset(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			calls, generated := 0, 0
			s := testSession(t, func(ctx context.Context, r *model.Request, o model.Options) (*model.Response, error) {
				calls++
				if o.Model != "chosen-model" || o.BaseURL != "http://model.invalid" {
					t.Errorf("configuration lost: %+v", o)
				}
				wantHistory := 0
				if calls == 2 {
					wantHistory = 2
				}
				if len(r.Messages) != wantHistory {
					t.Errorf("turn %d: history %v", calls, r.Messages)
				}
				found := false
				for _, tool := range r.Tools {
					if tool.Name == generateTool.Name {
						found = true
					}
				}
				if !found {
					t.Error("generation tool missing")
				}
				result := o.ToolHandler(ctx, model.ToolCall{Name: generateTool.Name, Input: map[string]any{"description": "notes"}})
				if result.Content != "created" {
					t.Errorf("tool result: %+v", result)
				}
				return &model.Response{Reply: "done"}, nil
			})
			s.stream = stream
			s.generate = func(context.Context, map[string]any) (string, error) { generated++; return "created", nil }
			for _, prompt := range []string{"first", "second"} {
				if err := s.ask(context.Background(), prompt); err != nil {
					t.Fatal(err)
				}
			}
			previous := s.local
			s.reset()
			if err := s.ask(context.Background(), "fresh"); err != nil {
				t.Fatal(err)
			}
			if s.local == previous || generated != 3 {
				t.Fatalf("reset/tool execution failed: %d", generated)
			}
		})
	}
}

func TestHarnessGuardsRepeatedGeneration(t *testing.T) {
	executed, refused := 0, 0
	s := testSession(t, func(ctx context.Context, _ *model.Request, o model.Options) (*model.Response, error) {
		for i := 0; i < 5; i++ {
			r := o.ToolHandler(ctx, model.ToolCall{Name: generateTool.Name, Input: map[string]any{"description": "notes"}})
			if r.Refused == model.RefusedLoop {
				refused++
			}
		}
		return &model.Response{Reply: "done"}, nil
	})
	s.generate = func(context.Context, map[string]any) (string, error) { executed++; return "created", nil }
	if err := s.ask(context.Background(), "build"); err != nil {
		t.Fatal(err)
	}
	if executed != 3 || refused != 2 {
		t.Fatalf("executed=%d refused=%d", executed, refused)
	}
}

type remoteClient struct {
	client.Client
	calls, streams int
}

func (c *remoteClient) Call(_ context.Context, _ client.Request, rsp any, _ ...client.CallOption) error {
	c.calls++
	rsp.(*bytes.Frame).Data = []byte(`{"reply":"done"}`)
	return nil
}
func (c *remoteClient) Stream(context.Context, client.Request, ...client.CallOption) (client.Stream, error) {
	c.streams++
	return nil, errors.New("stream interrupted")
}
func TestRemoteStreamingDoesNotReplay(t *testing.T) {
	for _, supported := range []bool{false, true} {
		t.Run(fmt.Sprint(supported), func(t *testing.T) {
			c := &remoteClient{Client: client.DefaultClient}
			s := &session{cl: c, stream: true, agents: map[string]agentInfo{"assistant": {Name: "assistant", Stream: supported}}}
			err := s.ask(context.Background(), "perform action")
			if supported {
				if err == nil || c.streams != 1 || c.calls != 0 {
					t.Fatalf("replayed stream: %v %+v", err, c)
				}
			} else if err != nil || c.calls != 1 || c.streams != 0 {
				t.Fatalf("legacy agent: %v %+v", err, c)
			}
		})
	}
}

func TestProcessCannotStartAfterCancellationOrCleanup(t *testing.T) {
	s := &session{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.startProcess(ctx, exec.Command("nonexistent-chat-test")); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	s.cleanup()
	if err := s.startProcess(context.Background(), exec.Command("nonexistent-chat-test")); err == nil || err.Error() != "chat session closed" {
		t.Fatalf("got %v", err)
	}
}

func TestGeneratedServiceDiscoveredOnNextTurn(t *testing.T) {
	var s *session
	turn := 0
	s = testSession(t, func(ctx context.Context, r *model.Request, o model.Options) (*model.Response, error) {
		turn++
		found := false
		for _, tool := range r.Tools {
			if tool.OriginalName == "notes.Notes.List" {
				found = true
			}
		}
		if found != (turn == 2) {
			t.Errorf("turn %d discovered notes=%v", turn, found)
		}
		if turn == 1 {
			o.ToolHandler(ctx, model.ToolCall{Name: generateTool.Name, Input: map[string]any{"description": "notes"}})
		}
		return &model.Response{Reply: "done"}, nil
	})
	s.generate = func(context.Context, map[string]any) (string, error) {
		return "created", s.reg.Register(&registry.Service{Name: "notes", Version: "1", Nodes: []*registry.Node{{Id: "one", Address: "127.0.0.1:1"}}, Endpoints: []*registry.Endpoint{{Name: "Notes.List"}}})
	}
	for _, prompt := range []string{"build notes", "list notes"} {
		if err := s.ask(context.Background(), prompt); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRouterUsesConfiguredHarness(t *testing.T) {
	s := testSession(t, func(ctx context.Context, r *model.Request, o model.Options) (*model.Response, error) {
		if o.Model != "chosen-model" || o.BaseURL != "http://model.invalid" {
			t.Error("router lost provider configuration")
		}
		for _, tool := range r.Tools {
			if tool.Name == generateTool.Name {
				t.Error("router exposes generation")
			}
		}
		result := o.ToolHandler(ctx, model.ToolCall{Name: "route_to_agent", Input: map[string]any{"agent": "first", "message": r.Prompt}})
		if result.Content == "" {
			t.Error("missing routed response")
		}
		return &model.Response{Reply: "done"}, nil
	})
	c := &remoteClient{Client: client.DefaultClient}
	s.cl = c
	s.agents = map[string]agentInfo{"first": {Name: "first"}, "second": {Name: "second"}}
	if err := s.ask(context.Background(), "perform action"); err != nil {
		t.Fatal(err)
	}
	if c.calls != 1 {
		t.Fatalf("calls=%d", c.calls)
	}
}

func TestChatDiscoversSingleAgentWithoutLocalModel(t *testing.T) {
	oldDir := store.DefaultDir
	store.DefaultDir = t.TempDir()
	defer func() { store.DefaultDir = oldDir }()
	for _, interactive := range []bool{false, true} {
		t.Run(fmt.Sprint(interactive), func(t *testing.T) {
			reg := registry.NewMemoryRegistry()
			if err := reg.Register(&registry.Service{Name: "assistant", Metadata: map[string]string{"type": "agent", "sessions": "v1"}, Nodes: []*registry.Node{{Id: "assistant-1", Address: "localhost:1234"}}}); err != nil {
				t.Fatal(err)
			}
			oldReg, oldClient, oldStdin := registry.DefaultRegistry, client.DefaultClient, os.Stdin
			remote := &remoteClient{Client: oldClient}
			registry.DefaultRegistry, client.DefaultClient = reg, remote
			t.Cleanup(func() { registry.DefaultRegistry, client.DefaultClient, os.Stdin = oldReg, oldClient, oldStdin })
			input, err := os.CreateTemp(t.TempDir(), "input")
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			if _, err := input.WriteString("Hello\nexit\n"); err != nil {
				t.Fatal(err)
			}
			if _, err := input.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
			os.Stdin = input
			flags := flag.NewFlagSet("chat", flag.ContinueOnError)
			// An unknown local provider proves the remote path needs no adapter.
			flags.String("provider", "unconfigured", "")
			flags.String("api_key", "", "")
			prompt := "Hello"
			if interactive {
				prompt = ""
			}
			flags.String("prompt", prompt, "")
			ctx := cli.NewContext(cli.NewApp(), flags, nil)
			if err := run(ctx); err != nil {
				t.Fatal(err)
			}
			if remote.calls != 1 {
				t.Fatalf("got %d calls, want 1", remote.calls)
			}
		})
	}
}

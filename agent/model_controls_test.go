package agent

import (
	"context"
	"io"
	"reflect"
	"testing"

	"go-micro.dev/v6/model"
)

func TestAgentModelControlsAcrossCallPaths(t *testing.T) {
	for _, mode := range []string{"ask", "stream_ask", "stream", "delegate"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			check := func(opts model.Options) {
				calls++
				if opts.MaxTokens != 1024 || opts.Effort != "low" || opts.Temperature == nil || *opts.Temperature != 0 || opts.BaseURL != "http://unused.test" {
					t.Errorf("options lost: %+v", opts)
				}
			}
			fakeGen = func(_ context.Context, opts model.Options, _ *model.Request) (*model.Response, error) {
				check(opts)
				return &model.Response{Reply: "ok"}, nil
			}
			fakeStream = func(_ context.Context, opts model.Options, _ *model.Request) (model.Stream, error) {
				check(opts)
				return &sliceStream{chunks: []string{"ok"}}, nil
			}
			defer func() { fakeGen = nil; fakeStream = nil }()
			a := newTestAgent(Name("controls"), BaseURL("http://unused.test"), MaxTokens(1024), Effort("low"), Temperature(0), MaxTools(1))
			ctx := context.Background()
			switch mode {
			case "ask":
				if _, err := a.Ask(ctx, "hello"); err != nil {
					t.Fatal(err)
				}
			case "stream_ask":
				s, err := a.StreamAsk(ctx, "hello")
				if err != nil {
					t.Fatal(err)
				}
				defer s.Close()
				for {
					_, err = s.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
				}
			case "stream":
				s, err := a.Stream(ctx, "hello")
				if err != nil {
					t.Fatal(err)
				}
				defer s.Close()
				for {
					_, err = s.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
				}
			case "delegate":
				result := a.toolHandler()(ctx, model.ToolCall{ID: "delegate", Name: "delegate", Input: map[string]any{"task": "hello"}})
				if result.Content == "" {
					t.Fatal("empty delegate result")
				}
			}
			if calls != 1 {
				t.Fatalf("provider calls=%d", calls)
			}
		})
	}
}

func TestMaxToolsDeterministicAcrossDiscoveryOrder(t *testing.T) {
	for _, names := range [][]string{{"zeta", "alpha", "beta"}, {"beta", "zeta", "alpha"}} {
		for _, limit := range []int{0, 2, 10} {
			opts := []Option{Name("cap"), Services(), MaxTools(limit)}
			for _, name := range names {
				opts = append(opts, WithTool(name, name, nil, func(context.Context, map[string]any) (string, error) { return "ok", nil }))
			}
			a := newTestAgent(opts...)
			tools, err := a.discoverTools()
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, tool := range tools {
				got = append(got, tool.Name)
			}
			if limit == 2 {
				if !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
					t.Fatalf("capped=%v", got)
				}
			} else if len(got) != len(names)+len(builtinTools()) {
				t.Fatalf("uncapped=%v", got)
			}
		}
	}
}

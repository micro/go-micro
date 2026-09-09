package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
	microcmd "go-micro.dev/v6/cmd"
)

func TestExamplesWayfindingIndexStaysLinked(t *testing.T) {
	root := filepath.Join("..", "..")
	files := map[string]string{}
	for _, name := range []string{"README.md", "examples/README.md", "examples/INDEX.md"} {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		files[name] = string(b)
	}

	for _, check := range []struct {
		file string
		want []string
	}{
		{
			file: "README.md",
			want: []string{"examples/INDEX.md", "examples/first-agent/", "examples/support/", "zero-to-hero.md"},
		},
		{
			file: "examples/README.md",
			want: []string{"./INDEX.md", "./first-agent/", "./support/", "./mcp/hello/", "./mcp/workflow/"},
		},
		{
			file: "examples/INDEX.md",
			want: []string{"go run ./examples/first-agent", "go run ./examples/support", "mcp/hello", "mcp/workflow", "flow-durable", "micro examples"},
		},
	} {
		for _, want := range check.want {
			if !strings.Contains(files[check.file], want) {
				t.Fatalf("%s missing %q", check.file, want)
			}
		}
	}
}

func TestExamplesCommandPointsAtWayfindingIndex(t *testing.T) {
	examples := commandByName(t, "examples")
	var out bytes.Buffer
	app := cli.NewApp()
	app.Writer = &out
	if err := examples.Action(cli.NewContext(app, nil, nil)); err != nil {
		t.Fatalf("micro examples failed: %v", err)
	}

	for _, want := range []string{
		"examples/INDEX.md",
		"go run ./examples/first-agent",
		"go run ./examples/support",
		"micro zero-to-hero",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("micro examples output missing %q:\n%s", want, out.String())
		}
	}

	_ = microcmd.DefaultCmd // keep this test coupled to the registered command package.
}

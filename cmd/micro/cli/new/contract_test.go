package new

import (
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/urfave/cli/v2"
)

// TestZeroToOneContract locks the documented getting-started path:
// `micro new helloworld` must produce an ordinary Go service that the Go
// toolchain can build, run long enough to start, and call through its generated
// handler. The generated module is pointed back at this checkout so the
// contract stays local and deterministic in CI.
//
// It shells out to `micro new` (which runs `go mod tidy`) and `go build`, so
// it needs the Go toolchain and module access; it is skipped under `-short`.
func TestZeroToOneContract(t *testing.T) {
	generated := generateService(t, "helloworld")

	for _, rel := range []string{"go.mod", "main.go", "handler/helloworld.go", "README.md", "Makefile"} {
		if _, err := os.Stat(filepath.Join(generated.dir, rel)); err != nil {
			t.Fatalf("generated file %s: %v", rel, err)
		}
	}

	generated.replaceModule(t)
	generated.build(t)
	generated.run(t)
	generated.call(t, "Alice", "Hello Alice")
}

// TestZeroToOneNoMCPContract keeps the MCP opt-out path honest. Some services
// intentionally run without the local MCP listener, but that variant must still
// satisfy the same 0→1 contract: scaffold, tidy, and build without additional
// toolchain dependencies.
func TestZeroToOneNoMCPContract(t *testing.T) {
	generated := generateService(t, "worker", "--no-mcp")

	main, err := os.ReadFile(filepath.Join(generated.dir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(main), "gateway/mcp") || strings.Contains(string(main), "WithMCP") {
		t.Fatalf("--no-mcp generated main.go with MCP wiring:\n%s", main)
	}

	generated.replaceModule(t)
	generated.build(t)
	generated.run(t)
	generated.call(t, "Bob", "Hello Bob")
}

type generatedService struct {
	dir      string
	repoRoot string
}

func generateService(t *testing.T, name string, args ...string) generatedService {
	t.Helper()

	if testing.Short() {
		t.Skip("contract test shells out to the Go toolchain; skipped with -short")
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	tmp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	set := flag.NewFlagSet("micro-new", flag.ContinueOnError)
	set.Bool("no-mcp", false, "")
	set.Bool("proto", false, "")
	set.String("template", "", "")
	set.String("prompt", "", "")
	set.String("provider", "", "")
	set.String("api_key", "", "")
	if err := set.Parse(append(args, name)); err != nil {
		t.Fatal(err)
	}
	ctx := cli.NewContext(cli.NewApp(), set, nil)

	if err := Run(ctx); err != nil {
		t.Fatalf("micro new %s %s: %v", strings.Join(args, " "), name, err)
	}

	return generatedService{dir: filepath.Join(tmp, name), repoRoot: repoRoot}
}

func (g generatedService) replaceModule(t *testing.T) {
	t.Helper()

	modPath := filepath.Join(g.dir, "go.mod")
	mod, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatal(err)
	}
	modText := strings.Replace(string(mod), "go-micro.dev/v6 latest", "go-micro.dev/v6 v6.0.0", 1)
	modText += "\nreplace go-micro.dev/v6 => " + filepath.ToSlash(g.repoRoot) + "\n"
	if err := os.WriteFile(modPath, []byte(modText), 0644); err != nil {
		t.Fatal(err)
	}
}

func (g generatedService) build(t *testing.T) {
	t.Helper()

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated service go build ./... failed: %v\n%s", err, out)
	}
}

func (g generatedService) run(t *testing.T) {
	t.Helper()

	bin := filepath.Join(g.dir, "service-contract")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = g.dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("generated service go build -o service-contract . failed: %v\n%s", err, out)
	}

	cmd := exec.Command(bin)
	cmd.Dir = g.dir
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Start(); err != nil {
		t.Fatalf("generated service failed to start: %v\n%s", err, out.String())
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		t.Fatalf("generated service exited early: %v\n%s", err, out.String())
	case <-time.After(2 * time.Second):
	}

	if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("failed to stop generated service: %v\n%s", err, out.String())
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("generated service did not stop after kill\n%s", out.String())
	}
}

func (g generatedService) call(t *testing.T, name, want string) {
	t.Helper()

	testPath := filepath.Join(g.dir, "handler", "contract_test.go")
	testSrc := `package handler

import (
	"context"
	"testing"
)

func TestGeneratedCallContract(t *testing.T) {
	rsp := new(Response)
	if err := New().Call(context.Background(), &Request{Name: "` + name + `"}, rsp); err != nil {
		t.Fatal(err)
	}
	if rsp.Msg != "` + want + `" {
		t.Fatalf("Call response = %q, want %q", rsp.Msg, "` + want + `")
	}
}
`
	if err := os.WriteFile(testPath, []byte(testSrc), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "test", "./handler", "-run", "TestGeneratedCallContract", "-count=1")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated service call contract failed: %v\n%s", err, out)
	}
}

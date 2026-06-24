package new

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

// TestZeroToOneContract locks the documented getting-started path:
// `micro new helloworld` must produce an ordinary Go service that can be
// tested by the Go toolchain. The generated module is pointed back at this
// checkout so the contract stays local and deterministic in CI.
func TestZeroToOneContract(t *testing.T) {
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
	if err := set.Parse([]string{"helloworld"}); err != nil {
		t.Fatal(err)
	}
	ctx := cli.NewContext(cli.NewApp(), set, nil)

	if err := Run(ctx); err != nil {
		t.Fatalf("micro new helloworld: %v", err)
	}

	serviceDir := filepath.Join(tmp, "helloworld")
	for _, rel := range []string{"go.mod", "main.go", "handler/helloworld.go", "README.md", "Makefile"} {
		if _, err := os.Stat(filepath.Join(serviceDir, rel)); err != nil {
			t.Fatalf("generated file %s: %v", rel, err)
		}
	}

	modPath := filepath.Join(serviceDir, "go.mod")
	mod, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatal(err)
	}
	modText := strings.Replace(string(mod), "go-micro.dev/v6 latest", "go-micro.dev/v6 v6.0.0", 1)
	modText += "\nreplace go-micro.dev/v6 => " + filepath.ToSlash(repoRoot) + "\n"
	if err := os.WriteFile(modPath, []byte(modText), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = serviceDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated service go test ./... failed: %v\n%s", err, out)
	}
}

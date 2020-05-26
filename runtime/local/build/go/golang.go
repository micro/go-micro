// Package golang is a go package manager
package golang

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/micro/go-micro/v2/runtime/local/build"
)

type Builder struct {
	Options build.Options
	Cmd     string
	Path    string
}

// whichGo locates the go command
func whichGo() string {
	// check GOROOT
	if gr := os.Getenv("GOROOT"); len(gr) > 0 {
		return filepath.Join(gr, "bin", "go")
	}

	// check path
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		bin := filepath.Join(p, "go")
		if _, err := os.Stat(bin); err == nil {
			return bin
		}
	}

	// best effort
	return "go"
}

func (g *Builder) Build(s *build.Source) (*build.Package, error) {
	binary := filepath.Join(g.Path, s.Repository.Name)
	source := filepath.Join(s.Repository.Path, s.Repository.Name)

	cmd := exec.Command(g.Cmd, "build", "-o", binary, source)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return &build.Package{
		Name:   s.Repository.Name,
		Path:   binary,
		Type:   "go",
		Source: s,
	}, nil
}

func (g *Builder) Clean(b *build.Package) error {
	binary := filepath.Join(b.Path, b.Name)
	return os.Remove(binary)
}

func NewBuild(opts ...build.Option) build.Builder {
	options := build.Options{
		Path: os.TempDir(),
	}
	for _, o := range opts {
		o(&options)
	}
	return &Builder{
		Options: options,
		Cmd:     whichGo(),
		Path:    options.Path,
	}
}

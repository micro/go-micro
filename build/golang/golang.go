// Package golang is a go package manager
package golang

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/micro/go-micro/v3/build"
)

type goBuild struct {
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

func (g *goBuild) Package(name string, src *build.Source) (*build.Package, error) {
	binary := filepath.Join(g.Path, name)
	source := src.Path

	cmd := exec.Command(g.Cmd, "build", "-o", binary, source)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return &build.Package{
		Name:   name,
		Path:   binary,
		Type:   g.String(),
		Source: src,
	}, nil
}

func (g *goBuild) Remove(b *build.Package) error {
	binary := filepath.Join(b.Path, b.Name)
	return os.Remove(binary)
}

func (g *goBuild) String() string {
	return "golang"
}

func NewBuild(opts ...build.Option) build.Build {
	options := build.Options{
		Path: os.TempDir(),
	}
	for _, o := range opts {
		o(&options)
	}
	return &goBuild{
		Options: options,
		Cmd:     whichGo(),
		Path:    options.Path,
	}
}

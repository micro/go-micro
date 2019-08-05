// Package golang is a go package manager
package golang

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/micro/go-micro/runtime/package"
)

type Packager struct {
	Options packager.Options
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

func (g *Packager) Compile(s *packager.Source) (*packager.Binary, error) {
	binary := filepath.Join(g.Path, s.Repository.Name)
	source := filepath.Join(s.Repository.Path, s.Repository.Name)

	cmd := exec.Command(g.Cmd, "build", "-o", binary, source)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return &packager.Binary{
		Name:   s.Repository.Name,
		Path:   binary,
		Type:   "go",
		Source: s,
	}, nil
}

func (g *Packager) Delete(b *packager.Binary) error {
	binary := filepath.Join(b.Path, b.Name)
	return os.Remove(binary)
}

func NewPackager(opts ...packager.Option) packager.Packager {
	options := packager.Options{
		Path: os.TempDir(),
	}
	for _, o := range opts {
		o(&options)
	}
	return &Packager{
		Options: options,
		Cmd:     whichGo(),
		Path:    options.Path,
	}
}

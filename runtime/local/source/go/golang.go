// Package golang is a source for Go
package golang

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/micro/go-micro/v2/runtime/local/source"
)

type Source struct {
	Options source.Options
	// Go Command
	Cmd  string
	Path string
}

func (g *Source) Fetch(url string) (*source.Repository, error) {
	purl := url

	if parts := strings.Split(url, "://"); len(parts) > 1 {
		purl = parts[len(parts)-1]
	}

	// name of repo
	name := filepath.Base(url)
	// local path of repo
	path := filepath.Join(g.Path, purl)
	args := []string{"get", "-d", url, path}

	cmd := exec.Command(g.Cmd, args...)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return &source.Repository{
		Name: name,
		Path: path,
		URL:  url,
	}, nil
}

// Commit is not yet supported
func (g *Source) Commit(r *source.Repository) error {
	return nil
}

func (g *Source) String() string {
	return "golang"
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

func NewSource(opts ...source.Option) source.Source {
	options := source.Options{
		Path: os.TempDir(),
	}
	for _, o := range opts {
		o(&options)
	}

	cmd := whichGo()
	path := options.Path

	// point of no return
	if len(cmd) == 0 {
		panic("Could not find Go executable")
	}

	return &Source{
		Options: options,
		Cmd:     cmd,
		Path:    path,
	}
}

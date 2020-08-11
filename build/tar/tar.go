// Package tar basically tarballs source code
package tar

import (
	"os"
	"path/filepath"

	"github.com/micro/go-micro/v3/build"
)

type tarBuild struct{}

func (t *tarBuild) Package(name string, src *build.Source) (*build.Package, error) {
	pkg := name + ".tar.gz"
	// path to the tarball
	path := filepath.Join(os.TempDir(), src.Path, pkg)

	// create a temp directory
	if err := os.MkdirAll(filepath.Join(os.TempDir(), src.Path), 0755); err != nil {
		return nil, err
	}

	if err := Compress(src.Path, path); err != nil {
		return nil, err
	}

	return &build.Package{
		Name:   name,
		Path:   path,
		Type:   t.String(),
		Source: src,
	}, nil
}

func (t *tarBuild) Remove(b *build.Package) error {
	return os.Remove(b.Path)
}

func (t *tarBuild) String() string {
	return "tar.gz"
}

func NewBuild(opts ...build.Option) build.Build {
	return new(tarBuild)
}

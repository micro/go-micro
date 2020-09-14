package build

import (
	"archive/tar"
	"path"
)

// LocalSource is source stored locally
type LocalSource struct {
	// Location of the source, e.g. /tmp/micro/foo-bar/latest
	Location string
}

// String to identify the local source
func (l *LocalSource) String() string {
	return l.Location
}

// GitSource is source in a remote git repository
type GitSource struct {
	// Repository containing the source, e.g. m3o/services
	Repository string
	// Remote containing the source, e.g. https://github.com/micro/micro
	Remote string
	// Folder the source exists within the repository, e.g. foo/api
	Folder string
}

// String returns the location of the git source
func (g *GitSource) String() string {
	return path.Join(g.Remote, g.Repository, g.Folder)
}

// TarSource is source contained within a tar archive
type TarSource struct {
	// Name of the source
	Name string
	// Reader to use to read the tar source from
	Reader tar.Reader
	// Folder within the tar which contains the source, e.g. foo/api
	Folder string
}

// String returns the name of the tar
func (t *TarSource) String() string {
	return t.Name
}

// Package file is a file source. Expected format is json
package file

import (
	"io"
	"io/fs"
	"os"

	"go-micro.dev/v4/config/source"
)

type file struct {
	fs   fs.FS
	path string
	opts source.Options
}

var (
	DefaultPath = "config.json"
)

func (f *file) Read() (*source.ChangeSet, error) {
	var fh fs.File
	var err error

	if f.fs != nil {
		fh, err = f.fs.Open(f.path)
	} else {
		fh, err = os.Open(f.path)
	}

	if err != nil {
		return nil, err
	}
	defer fh.Close()
	b, err := io.ReadAll(fh)
	if err != nil {
		return nil, err
	}
	info, err := fh.Stat()
	if err != nil {
		return nil, err
	}

	cs := &source.ChangeSet{
		Format:    format(f.path, f.opts.Encoder),
		Source:    f.String(),
		Timestamp: info.ModTime(),
		Data:      b,
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (f *file) String() string {
	return "file"
}

func (f *file) Watch() (source.Watcher, error) {
	// do not watch if fs.FS instance is provided
	if f.fs != nil {
		return source.NewNoopWatcher()
	}

	if _, err := os.Stat(f.path); err != nil {
		return nil, err
	}
	return newWatcher(f)
}

func (f *file) Write(cs *source.ChangeSet) error {
	return nil
}

func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)

	fs, _ := options.Context.Value(fsKey{}).(fs.FS)

	path := DefaultPath
	f, ok := options.Context.Value(filePathKey{}).(string)
	if ok {
		path = f
	}
	return &file{opts: options, fs: fs, path: path}
}

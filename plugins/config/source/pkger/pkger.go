package pkger

import (
	"io/ioutil"

	"github.com/markbates/pkger"
	"github.com/micro/go-micro/v2/config/source"
)

type file struct {
	path string
	opts source.Options
}

var (
	DefaultPath = "/config.yaml"
)

func (f *file) Read() (*source.ChangeSet, error) {
	fh, err := pkger.Open(f.path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	b, err := ioutil.ReadAll(fh)
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

func (f *file) Watch() (source.Watcher, error) {
	return source.NewNoopWatcher()
}

// Write is unsupported
func (f *file) Write(cs *source.ChangeSet) error {
	return nil
}

func (f *file) String() string {
	return "pkger"
}

func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)
	path := DefaultPath
	f, ok := options.Context.Value(pkgerPathKey{}).(string)
	if ok {
		path = f
	}
	return &file{opts: options, path: path}
}

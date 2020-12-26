// package runtimevar is the source for gocloud.dev/runtimevar
package runtimevar

import (
	"context"
	"fmt"
	"sync"

	"github.com/micro/go-micro/v2/config/source"
	"gocloud.dev/runtimevar"
)

type rvSource struct {
	opts source.Options

	sync.Mutex
	v *runtimevar.Variable
}

func (rv *rvSource) Read() (*source.ChangeSet, error) {
	s, err := rv.v.Latest(context.Background())
	if err != nil {
		return nil, err
	}

	// assuming value is bytes
	b, err := rv.opts.Encoder.Encode(s.Value.([]byte))
	if err != nil {
		return nil, fmt.Errorf("error reading source: %v", err)
	}

	cs := &source.ChangeSet{
		Timestamp: s.UpdateTime,
		Format:    rv.opts.Encoder.String(),
		Source:    rv.String(),
		Data:      b,
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (rv *rvSource) Watch() (source.Watcher, error) {
	return newWatcher(rv.String(), rv.v, rv.opts)
}

// Write is unsupported
func (rv *rvSource) Write(cs *source.ChangeSet) error {
	return nil
}

func (rv *rvSource) String() string {
	return "runtimevar"
}

func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)

	v, ok := options.Context.Value(variableKey{}).(*runtimevar.Variable)
	if !ok {
		// nooooooo
		panic("runtimevar.Variable required")
	}

	return &rvSource{
		opts: options,
		v:    v,
	}
}

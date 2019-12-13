package sync

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/micro/go-micro/store"
	ckv "github.com/micro/go-micro/store/etcd"
	lock "github.com/micro/go-micro/sync/lock/etcd"
)

type syncMap struct {
	opts Options
}

func ekey(k interface{}) string {
	b, _ := json.Marshal(k)
	return base64.StdEncoding.EncodeToString(b)
}

func (m *syncMap) Read(key, val interface{}) error {
	if key == nil {
		return fmt.Errorf("key is nil")
	}

	kstr := ekey(key)

	// lock
	if err := m.opts.Lock.Acquire(kstr); err != nil {
		return err
	}
	defer m.opts.Lock.Release(kstr)

	// get key
	kval, err := m.opts.Store.Read(kstr)
	if err != nil {
		return err
	}

	if len(kval) == 0 {
		return store.ErrNotFound
	}

	// decode value
	return json.Unmarshal(kval[0].Value, val)
}

func (m *syncMap) Write(key, val interface{}) error {
	if key == nil {
		return fmt.Errorf("key is nil")
	}

	kstr := ekey(key)

	// lock
	if err := m.opts.Lock.Acquire(kstr); err != nil {
		return err
	}
	defer m.opts.Lock.Release(kstr)

	// encode value
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}

	// set key
	return m.opts.Store.Write(&store.Record{
		Key:   kstr,
		Value: b,
	})
}

func (m *syncMap) Delete(key interface{}) error {
	if key == nil {
		return fmt.Errorf("key is nil")
	}

	kstr := ekey(key)

	// lock
	if err := m.opts.Lock.Acquire(kstr); err != nil {
		return err
	}
	defer m.opts.Lock.Release(kstr)
	return m.opts.Store.Delete(kstr)
}

func (m *syncMap) Iterate(fn func(key, val interface{}) error) error {
	keyvals, err := m.opts.Store.List()
	if err != nil {
		return err
	}

	sort.Slice(keyvals, func(i, j int) bool {
		return keyvals[i].Key < keyvals[j].Key
	})

	for _, keyval := range keyvals {
		// lock
		if err := m.opts.Lock.Acquire(keyval.Key); err != nil {
			return err
		}
		// unlock
		defer m.opts.Lock.Release(keyval.Key)

		// unmarshal value
		var val interface{}

		if len(keyval.Value) > 0 && keyval.Value[0] == '{' {
			if err := json.Unmarshal(keyval.Value, &val); err != nil {
				return err
			}
		} else {
			val = keyval.Value
		}

		// exec func
		if err := fn(keyval.Key, val); err != nil {
			return err
		}

		// save val
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}

		// no save
		if i := bytes.Compare(keyval.Value, b); i == 0 {
			return nil
		}

		// set key
		if err := m.opts.Store.Write(&store.Record{
			Key:   keyval.Key,
			Value: b,
		}); err != nil {
			return err
		}
	}

	return nil
}

func NewMap(opts ...Option) Map {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	if options.Lock == nil {
		options.Lock = lock.NewLock()
	}

	if options.Store == nil {
		options.Store = ckv.NewStore()
	}

	return &syncMap{
		opts: options,
	}
}

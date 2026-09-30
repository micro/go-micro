package runstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-micro.dev/v6/model"
	bolt "go.etcd.io/bbolt"
	"sort"
	"sync"
	"time"
)

// Locker excludes simultaneous execution of a run. Implementations must fence
// other owners, including other processes, until release or owner termination.
type Locker interface {
	Lock(context.Context, string) (func(), error)
}

// Journal is a single-host checkpoint backend. Bolt holds an exclusive file
// lock for its lifetime; another host cannot open the same journal concurrently.
// It is not a distributed lease service. Close only after all runs stop.
type Journal struct {
	db    *bolt.DB
	mu    sync.Mutex
	locks map[string]chan struct{}
}

func OpenJournal(path string) (*Journal, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	return &Journal{db: db, locks: make(map[string]chan struct{})}, nil
}
func (j *Journal) Close() error { return j.db.Close() }
func (j *Journal) Lock(ctx context.Context, id string) (func(), error) {
	j.mu.Lock()
	ch := j.locks[id]
	if ch == nil {
		ch = make(chan struct{}, 1)
		j.locks[id] = ch
	}
	j.mu.Unlock()
	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (j *Journal) Save(ctx context.Context, r Run) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.SchemaVersion = 1
	r.Updated = time.Now()
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return j.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("runs"))
		if err != nil {
			return err
		}
		return b.Put([]byte(r.ID), data)
	})
}
func (j *Journal) Load(ctx context.Context, id string) (Run, bool, error) {
	if err := ctx.Err(); err != nil {
		return Run{}, false, err
	}
	var r Run
	ok := false
	err := j.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("runs"))
		if b == nil {
			return nil
		}
		data := b.Get([]byte(id))
		if data == nil {
			return nil
		}
		ok = true
		return decode(data, &r)
	})
	return r, ok, err
}
func (j *Journal) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return j.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("runs"))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(id))
	})
}
func (j *Journal) List(ctx context.Context) ([]Run, error) {
	var runs []Run
	err := j.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("runs"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, data []byte) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			var r Run
			if err := decode(data, &r); err != nil {
				return err
			}
			runs = append(runs, r)
			return nil
		})
	})
	sort.SliceStable(runs, func(i, j int) bool {
		if runs[i].Started.Equal(runs[j].Started) {
			return runs[i].ID < runs[j].ID
		}
		return runs[i].Started.Before(runs[j].Started)
	})
	return runs, err
}
func decode(data []byte, r *Run) error {
	if err := json.Unmarshal(data, r); err != nil {
		return err
	}
	if r.SchemaVersion > 1 || r.SchemaVersion < 0 {
		return fmt.Errorf("unsupported run schema %d", r.SchemaVersion)
	}
	return nil
}

var ErrLimit = model.ErrLimit
var ErrAmbiguous = errors.New("interrupted operation requires reconciliation or idempotent replay")

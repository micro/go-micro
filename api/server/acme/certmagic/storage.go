package certmagic

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/mholt/certmagic"
	"github.com/micro/go-micro/store"
	"github.com/micro/go-micro/sync/lock"
)

// File represents a "File" that will be stored in store.Store - the contents and last modified time
type File struct {
	// last modified time
	LastModified time.Time
	// Contents
	Contents []byte
}

// storage is an implementation of certmagic.Storage using micro's sync.Map and store.Store interfaces.
// As certmagic storage expects a filesystem (with stat() abilities) we have to implement
// the bare minimum of metadata.
type storage struct {
	lock  lock.Lock
	store store.Store
}

func (s *storage) Lock(key string) error {
	return s.lock.Acquire(key, lock.TTL(10*time.Minute))
}

func (s *storage) Unlock(key string) error {
	return s.lock.Release(key)
}

func (s *storage) Store(key string, value []byte) error {
	f := File{
		LastModified: time.Now(),
		Contents:     value,
	}
	buf := &bytes.Buffer{}
	e := gob.NewEncoder(buf)
	if err := e.Encode(f); err != nil {
		return err
	}
	r := &store.Record{
		Key:   key,
		Value: buf.Bytes(),
	}
	return s.store.Write(r)
}

func (s *storage) Load(key string) ([]byte, error) {
	if !s.Exists(key) {
		return nil, certmagic.ErrNotExist(errors.New(key + " doesn't exist"))
	}
	records, err := s.store.Read(key)
	if err != nil {
		return nil, err
	}
	if len(records) != 1 {
		return nil, fmt.Errorf("ACME Storage: multiple records matched key %s", key)
	}
	b := bytes.NewBuffer(records[0].Value)
	d := gob.NewDecoder(b)
	var f File
	err = d.Decode(&f)
	if err != nil {
		return nil, err
	}
	return f.Contents, nil
}

func (s *storage) Delete(key string) error {
	return s.store.Delete(key)
}

func (s *storage) Exists(key string) bool {
	_, err := s.store.Read(key)
	if err != nil {
		return false
	}
	return true
}

func (s *storage) List(prefix string, recursive bool) ([]string, error) {
	records, err := s.store.Sync()
	if err != nil {
		return nil, err
	}
	var results []string
	for _, r := range records {
		if strings.HasPrefix(r.Key, prefix) {
			results = append(results, r.Key)
		}
	}
	if recursive {
		return results, nil
	}
	keysMap := make(map[string]bool)
	for _, key := range results {
		dir := strings.Split(strings.TrimPrefix(key, prefix+"/"), "/")
		keysMap[dir[0]] = true
	}
	results = make([]string, 0)
	for k := range keysMap {
		results = append(results, path.Join(prefix, k))
	}
	return results, nil
}

func (s *storage) Stat(key string) (certmagic.KeyInfo, error) {
	records, err := s.store.Read(key)
	if err != nil {
		return certmagic.KeyInfo{}, err
	}
	if len(records) != 1 {
		return certmagic.KeyInfo{}, fmt.Errorf("ACME Storage: multiple records matched key %s", key)
	}
	b := bytes.NewBuffer(records[0].Value)
	d := gob.NewDecoder(b)
	var f File
	err = d.Decode(&f)
	if err != nil {
		return certmagic.KeyInfo{}, err
	}
	return certmagic.KeyInfo{
		Key:        key,
		Modified:   f.LastModified,
		Size:       int64(len(f.Contents)),
		IsTerminal: false,
	}, nil
}

// NewStorage returns a certmagic.Storage backed by a go-micro/lock and go-micro/store
func NewStorage(lock lock.Lock, store store.Store) certmagic.Storage {
	return &storage{
		lock:  lock,
		store: store,
	}
}

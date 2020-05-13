package certmagic

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/micro/go-micro/v2/store"
	"github.com/micro/go-micro/v2/sync"
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
	lock  sync.Sync
	store store.Store
}

func (s *storage) Lock(key string) error {
	return s.lock.Lock(key, sync.LockTTL(10*time.Minute))
}

func (s *storage) Unlock(key string) error {
	return s.lock.Unlock(key)
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
	if _, err := s.store.Read(key); err != nil {
		return false
	}
	return true
}

func (s *storage) List(prefix string, recursive bool) ([]string, error) {
	keys, err := s.store.List()
	if err != nil {
		return nil, err
	}

	//nolint:prealloc
	var results []string
	for _, k := range keys {
		if strings.HasPrefix(k, prefix) {
			results = append(results, k)
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
func NewStorage(lock sync.Sync, store store.Store) certmagic.Storage {
	return &storage{
		lock:  lock,
		store: store,
	}
}

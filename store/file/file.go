// Package local is a file system backed store
package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/store"
	bolt "go.etcd.io/bbolt"
)

var (
	// DefaultDatabase is the namespace that the bbolt store
	// will use if no namespace is provided.
	DefaultDatabase = "micro"
	// DefaultTable when none is specified
	DefaultTable = "micro"
	// DefaultDir is the default directory for bbolt files
	DefaultDir = filepath.Join(os.TempDir(), "micro", "store")

	// bucket used for data storage
	dataBucket = "data"
)

// NewStore returns a memory store
func NewStore(opts ...store.Option) store.Store {
	s := &fileStore{
		handles: make(map[string]*fileHandle),
	}
	s.init(opts...)
	return s
}

type fileStore struct {
	options store.Options
	dir     string

	// the database handle
	sync.RWMutex
	handles map[string]*fileHandle
}

type fileHandle struct {
	key string
	db  *bolt.DB
}

// record stored by us
type record struct {
	Key       string
	Value     []byte
	Metadata  map[string]interface{}
	ExpiresAt time.Time
}

func key(database, table string) string {
	return database + ":" + table
}

func (m *fileStore) delete(fd *fileHandle, key string) error {
	return fd.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dataBucket))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(key))
	})
}

func (m *fileStore) init(opts ...store.Option) error {
	for _, o := range opts {
		o(&m.options)
	}

	if m.options.Database == "" {
		m.options.Database = DefaultDatabase
	}

	if m.options.Table == "" {
		// bbolt requires bucketname to not be empty
		m.options.Table = DefaultTable
	}

	// create a directory /tmp/micro
	dir := filepath.Join(DefaultDir, m.options.Database)
	// Ignoring this as the folder might exist.
	// Reads/Writes updates will return with sensible error messages
	// about the dir not existing in case this cannot create the path anyway
	os.MkdirAll(dir, 0700)

	return nil
}

func (f *fileStore) getDB(database, table string) (*fileHandle, error) {
	if len(database) == 0 {
		database = f.options.Database
	}
	if len(table) == 0 {
		table = f.options.Table
	}

	k := key(database, table)
	f.RLock()
	fd, ok := f.handles[k]
	f.RUnlock()

	// return the file handle
	if ok {
		return fd, nil
	}

	// double check locking
	f.Lock()
	defer f.Unlock()
	if fd, ok := f.handles[k]; ok {
		return fd, nil
	}

	// create a directory /tmp/micro
	dir := filepath.Join(DefaultDir, database)
	// create the database handle
	fname := table + ".db"
	// make the dir
	os.MkdirAll(dir, 0700)
	// database path
	dbPath := filepath.Join(dir, fname)

	// create new db handle
	// Bolt DB only allows one process to open the file R/W so make sure we're doing this under a lock
	db, err := bolt.Open(dbPath, 0700, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}
	fd = &fileHandle{
		key: k,
		db:  db,
	}
	f.handles[k] = fd

	return fd, nil
}

func (m *fileStore) list(fd *fileHandle, limit, offset uint) []string {
	var allItems []string

	fd.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dataBucket))
		// nothing to read
		if b == nil {
			return nil
		}

		// @todo very inefficient
		if err := b.ForEach(func(k, v []byte) error {
			storedRecord := &record{}

			if err := json.Unmarshal(v, storedRecord); err != nil {
				return err
			}

			if !storedRecord.ExpiresAt.IsZero() {
				if storedRecord.ExpiresAt.Before(time.Now()) {
					return nil
				}
			}

			allItems = append(allItems, string(k))

			return nil
		}); err != nil {
			return err
		}

		return nil
	})

	allKeys := make([]string, len(allItems))

	for i, k := range allItems {
		allKeys[i] = k
	}

	if limit != 0 || offset != 0 {
		sort.Slice(allKeys, func(i, j int) bool { return allKeys[i] < allKeys[j] })
		min := func(i, j uint) uint {
			if i < j {
				return i
			}
			return j
		}
		return allKeys[offset:min(limit, uint(len(allKeys)))]
	}

	return allKeys
}

func (m *fileStore) get(fd *fileHandle, k string) (*store.Record, error) {
	var value []byte

	fd.db.View(func(tx *bolt.Tx) error {
		// @todo this is still very experimental...
		b := tx.Bucket([]byte(dataBucket))
		if b == nil {
			return nil
		}

		value = b.Get([]byte(k))
		return nil
	})

	if value == nil {
		return nil, store.ErrNotFound
	}

	storedRecord := &record{}

	if err := json.Unmarshal(value, storedRecord); err != nil {
		return nil, err
	}

	newRecord := &store.Record{}
	newRecord.Key = storedRecord.Key
	newRecord.Value = storedRecord.Value
	newRecord.Metadata = make(map[string]interface{})

	for k, v := range storedRecord.Metadata {
		newRecord.Metadata[k] = v
	}

	if !storedRecord.ExpiresAt.IsZero() {
		if storedRecord.ExpiresAt.Before(time.Now()) {
			return nil, store.ErrNotFound
		}
		newRecord.Expiry = time.Until(storedRecord.ExpiresAt)
	}

	return newRecord, nil
}

func (m *fileStore) set(fd *fileHandle, r *store.Record) error {
	// copy the incoming record and then
	// convert the expiry in to a hard timestamp
	item := &record{}
	item.Key = r.Key
	item.Value = r.Value
	item.Metadata = make(map[string]interface{})

	if r.Expiry != 0 {
		item.ExpiresAt = time.Now().Add(r.Expiry)
	}

	for k, v := range r.Metadata {
		item.Metadata[k] = v
	}

	// marshal the data
	data, _ := json.Marshal(item)

	return fd.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dataBucket))
		if b == nil {
			var err error
			b, err = tx.CreateBucketIfNotExists([]byte(dataBucket))
			if err != nil {
				return err
			}
		}
		return b.Put([]byte(r.Key), data)
	})
}

func (f *fileStore) Close() error {
	f.Lock()
	defer f.Unlock()
	for k, v := range f.handles {
		v.db.Close()
		delete(f.handles, k)
	}
	return nil
}

func (f *fileStore) Init(opts ...store.Option) error {
	return f.init(opts...)
}

func (m *fileStore) Delete(key string, opts ...store.DeleteOption) error {
	var deleteOptions store.DeleteOptions
	for _, o := range opts {
		o(&deleteOptions)
	}

	fd, err := m.getDB(deleteOptions.Database, deleteOptions.Table)
	if err != nil {
		return err
	}

	return m.delete(fd, key)
}

func (m *fileStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	var readOpts store.ReadOptions
	for _, o := range opts {
		o(&readOpts)
	}

	fd, err := m.getDB(readOpts.Database, readOpts.Table)
	if err != nil {
		return nil, err
	}

	var keys []string

	// Handle Prefix / suffix
	// TODO: do range scan here rather than listing all keys
	if readOpts.Prefix || readOpts.Suffix {
		// list the keys
		k := m.list(fd, readOpts.Limit, readOpts.Offset)

		// check for prefix and suffix
		for _, v := range k {
			if readOpts.Prefix && !strings.HasPrefix(v, key) {
				continue
			}
			if readOpts.Suffix && !strings.HasSuffix(v, key) {
				continue
			}
			keys = append(keys, v)
		}
	} else {
		keys = []string{key}
	}

	var results []*store.Record

	for _, k := range keys {
		r, err := m.get(fd, k)
		if err != nil {
			return results, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (m *fileStore) Write(r *store.Record, opts ...store.WriteOption) error {
	var writeOpts store.WriteOptions
	for _, o := range opts {
		o(&writeOpts)
	}

	fd, err := m.getDB(writeOpts.Database, writeOpts.Table)
	if err != nil {
		return err
	}

	if len(opts) > 0 {
		// Copy the record before applying options, or the incoming record will be mutated
		newRecord := store.Record{}
		newRecord.Key = r.Key
		newRecord.Value = r.Value
		newRecord.Metadata = make(map[string]interface{})
		newRecord.Expiry = r.Expiry

		if !writeOpts.Expiry.IsZero() {
			newRecord.Expiry = time.Until(writeOpts.Expiry)
		}
		if writeOpts.TTL != 0 {
			newRecord.Expiry = writeOpts.TTL
		}

		for k, v := range r.Metadata {
			newRecord.Metadata[k] = v
		}

		return m.set(fd, &newRecord)
	}

	return m.set(fd, r)
}

func (m *fileStore) Options() store.Options {
	return m.options
}

func (m *fileStore) List(opts ...store.ListOption) ([]string, error) {
	var listOptions store.ListOptions

	for _, o := range opts {
		o(&listOptions)
	}

	fd, err := m.getDB(listOptions.Database, listOptions.Table)
	if err != nil {
		return nil, err
	}

	// TODO apply prefix/suffix in range query
	allKeys := m.list(fd, listOptions.Limit, listOptions.Offset)

	if len(listOptions.Prefix) > 0 {
		var prefixKeys []string
		for _, k := range allKeys {
			if strings.HasPrefix(k, listOptions.Prefix) {
				prefixKeys = append(prefixKeys, k)
			}
		}
		allKeys = prefixKeys
	}

	if len(listOptions.Suffix) > 0 {
		var suffixKeys []string
		for _, k := range allKeys {
			if strings.HasSuffix(k, listOptions.Suffix) {
				suffixKeys = append(suffixKeys, k)
			}
		}
		allKeys = suffixKeys
	}

	return allKeys, nil
}

func (m *fileStore) String() string {
	return "file"
}

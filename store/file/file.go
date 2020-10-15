// Package local is a file system backed store
package file

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/micro/go-micro/v3/store"
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

// NewStore returns a file store
func NewStore(opts ...store.Option) store.Store {
	s := &fileStore{}
	s.init(opts...)
	return s
}

type fileStore struct {
	options store.Options
	dir     string
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

func (m *fileStore) delete(db *bolt.DB, key string) error {
	return db.Update(func(tx *bolt.Tx) error {
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

	// Ignoring this as the folder might exist.
	// Reads/Writes updates will return with sensible error messages
	// about the dir not existing in case this cannot create the path anyway
	dir := m.getDir(m.options.Database)
	os.MkdirAll(dir, 0700)
	return nil
}

// getDir returns the directory which should contain the files for a databases
func (m *fileStore) getDir(db string) string {
	// get the directory option from the context
	var directory string
	if m.options.Context != nil {
		fd, ok := m.options.Context.Value(dirKey{}).(string)
		if ok {
			directory = fd
		}
	}
	if len(directory) == 0 {
		directory = DefaultDir
	}

	// construct the directory, e.g. /tmp/micro
	return filepath.Join(directory, db)
}

func (f *fileStore) getDB(database, table string) (*bolt.DB, error) {
	if len(database) == 0 {
		database = f.options.Database
	}
	if len(table) == 0 {
		table = f.options.Table
	}

	// create a directory /tmp/micro
	dir := f.getDir(database)
	// create the database handle
	fname := table + ".db"
	// make the dir
	os.MkdirAll(dir, 0700)
	// database path
	dbPath := filepath.Join(dir, fname)

	// create new db handle
	// Bolt DB only allows one process to open the file R/W so make sure we're doing this under a lock
	return bolt.Open(dbPath, 0700, &bolt.Options{Timeout: 5 * time.Second})
}

func (m *fileStore) list(db *bolt.DB, limit, offset uint, prefix, suffix string) []string {

	var keys []string

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dataBucket))
		// nothing to read
		if b == nil {
			return nil
		}
		c := b.Cursor()
		var k, v []byte
		var cont func(k []byte) bool

		if prefix != "" {
			// for prefix we can speed up the search, not for suffix though :(
			k, v = c.Seek([]byte(prefix))
			cont = func(k []byte) bool {
				return bytes.HasPrefix(k, []byte(prefix))
			}
		} else {
			k, v = c.First()
			cont = func(k []byte) bool {
				return true
			}
		}

		for ; k != nil && cont(k); k, v = c.Next() {
			storedRecord := &record{}

			if err := json.Unmarshal(v, storedRecord); err != nil {
				return err
			}
			if !storedRecord.ExpiresAt.IsZero() {
				if storedRecord.ExpiresAt.Before(time.Now()) {
					continue
				}
			}
			if suffix != "" && !bytes.HasSuffix(k, []byte(suffix)) {
				continue
			}
			if offset > 0 {
				offset--
				continue
			}
			keys = append(keys, string(k))
			// this check still works if no limit was passed to begin with, you'll just end up with large -ve value
			if limit == 1 {
				break
			}
			limit--

		}
		return nil
	})

	return keys
}

func (m *fileStore) get(db *bolt.DB, k string) (*store.Record, error) {
	var value []byte

	db.View(func(tx *bolt.Tx) error {
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

func (m *fileStore) set(db *bolt.DB, r *store.Record) error {
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

	return db.Update(func(tx *bolt.Tx) error {
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

	db, err := m.getDB(deleteOptions.Database, deleteOptions.Table)
	if err != nil {
		return err
	}
	defer db.Close()

	return m.delete(db, key)
}

func (m *fileStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	var readOpts store.ReadOptions
	for _, o := range opts {
		o(&readOpts)
	}

	db, err := m.getDB(readOpts.Database, readOpts.Table)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var keys []string

	// Handle Prefix / suffix
	if readOpts.Prefix || readOpts.Suffix {
		prefix := ""
		if readOpts.Prefix {
			prefix = key
		}
		suffix := ""
		if readOpts.Suffix {
			suffix = key
		}
		// list the keys
		keys = m.list(db, readOpts.Limit, readOpts.Offset, prefix, suffix)
	} else {
		keys = []string{key}
	}

	var results []*store.Record

	for _, k := range keys {
		r, err := m.get(db, k)
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

	db, err := m.getDB(writeOpts.Database, writeOpts.Table)
	if err != nil {
		return err
	}
	defer db.Close()

	if len(opts) > 0 {
		// Copy the record before applying options, or the incoming record will be mutated
		newRecord := store.Record{}
		newRecord.Key = r.Key
		newRecord.Value = r.Value
		newRecord.Metadata = make(map[string]interface{})
		newRecord.Expiry = r.Expiry

		for k, v := range r.Metadata {
			newRecord.Metadata[k] = v
		}

		return m.set(db, &newRecord)
	}

	return m.set(db, r)
}

func (m *fileStore) Options() store.Options {
	return m.options
}

func (m *fileStore) List(opts ...store.ListOption) ([]string, error) {
	var listOptions store.ListOptions

	for _, o := range opts {
		o(&listOptions)
	}

	db, err := m.getDB(listOptions.Database, listOptions.Table)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	allKeys := m.list(db, listOptions.Limit, listOptions.Offset, listOptions.Prefix, listOptions.Suffix)

	return allKeys, nil
}

func (m *fileStore) String() string {
	return "file"
}

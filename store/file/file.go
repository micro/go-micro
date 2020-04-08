// Package local is a file system backed store
package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/micro/go-micro/v2/store"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

var (
	// DefaultDatabase is the namespace that the bbolt store
	// will use if no namespace is provided.
	DefaultDatabase = "micro"
	// DefaultTable when none is specified
	DefaultTable = "micro"
	// DefaultDir is the default directory for bbolt files
	DefaultDir = os.TempDir()
)

// NewStore returns a memory store
func NewStore(opts ...store.Option) store.Store {
	s := &fileStore{}
	s.init(opts...)
	return s
}

type fileStore struct {
	options  store.Options
	dir      string
	fileName string
	dbPath   string
	// the database handle
	db *bolt.DB
}

// record stored by us
type record struct {
	Key       string
	Value     []byte
	ExpiresAt time.Time
}

func (m *fileStore) delete(key string) error {
	return m.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(m.options.Table))
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
	dir := filepath.Join(DefaultDir, "micro")
	// create the database handle
	fname := m.options.Database + ".db"
	// Ignoring this as the folder might exist.
	// Reads/Writes updates will return with sensible error messages
	// about the dir not existing in case this cannot create the path anyway
	_ = os.Mkdir(dir, 0700)

	m.dir = dir
	m.fileName = fname
	m.dbPath = filepath.Join(dir, fname)

	// close existing handle
	if m.db != nil {
		m.db.Close()
	}

	// create new db handle
	db, err := bolt.Open(m.dbPath, 0700, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return err
	}

	// set the new db
	m.db = db

	// create the table
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(m.options.Table))
		return err
	})
}

func (m *fileStore) list(limit, offset uint) []string {
	var allItems []string

	m.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(m.options.Table))
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

func (m *fileStore) get(k string) (*store.Record, error) {
	var value []byte

	m.db.View(func(tx *bolt.Tx) error {
		// @todo this is still very experimental...
		b := tx.Bucket([]byte(m.options.Table))
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

	if !storedRecord.ExpiresAt.IsZero() {
		if storedRecord.ExpiresAt.Before(time.Now()) {
			return nil, store.ErrNotFound
		}
		newRecord.Expiry = time.Until(storedRecord.ExpiresAt)
	}

	return newRecord, nil
}

func (m *fileStore) set(r *store.Record) error {
	// copy the incoming record and then
	// convert the expiry in to a hard timestamp
	item := &record{}
	item.Key = r.Key
	item.Value = r.Value
	if r.Expiry != 0 {
		item.ExpiresAt = time.Now().Add(r.Expiry)
	}

	// marshal the data
	data, _ := json.Marshal(item)

	return m.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(m.options.Table))
		if b == nil {
			var err error
			b, err = tx.CreateBucketIfNotExists([]byte(m.options.Table))
			if err != nil {
				return err
			}
		}
		return b.Put([]byte(r.Key), data)
	})
}

func (m *fileStore) Init(opts ...store.Option) error {
	return m.init(opts...)
}

func (m *fileStore) Delete(key string, opts ...store.DeleteOption) error {
	deleteOptions := store.DeleteOptions{}
	for _, o := range opts {
		o(&deleteOptions)
	}
	return m.delete(key)
}

func (m *fileStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	readOpts := store.ReadOptions{}
	for _, o := range opts {
		o(&readOpts)
	}

	var keys []string

	// Handle Prefix / suffix
	// TODO: do range scan here rather than listing all keys
	if readOpts.Prefix || readOpts.Suffix {
		var opts []store.ListOption
		if readOpts.Prefix {
			opts = append(opts, store.ListPrefix(key))
		}
		if readOpts.Suffix {
			opts = append(opts, store.ListSuffix(key))
		}

		opts = append(opts, store.ListLimit(readOpts.Limit))
		opts = append(opts, store.ListOffset(readOpts.Offset))

		k, err := m.List(opts...)
		if err != nil {
			return nil, errors.Wrap(err, "FileStore: Read couldn't List()")
		}

		keys = k
	} else {
		keys = []string{key}
	}

	var results []*store.Record

	for _, k := range keys {
		r, err := m.get(k)
		if err != nil {
			return results, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (m *fileStore) Write(r *store.Record, opts ...store.WriteOption) error {
	writeOpts := store.WriteOptions{}
	for _, o := range opts {
		o(&writeOpts)
	}

	if len(opts) > 0 {
		// Copy the record before applying options, or the incoming record will be mutated
		newRecord := store.Record{}
		newRecord.Key = r.Key
		newRecord.Value = r.Value
		newRecord.Expiry = r.Expiry

		if !writeOpts.Expiry.IsZero() {
			newRecord.Expiry = time.Until(writeOpts.Expiry)
		}
		if writeOpts.TTL != 0 {
			newRecord.Expiry = writeOpts.TTL
		}

		return m.set(&newRecord)
	}

	return m.set(r)
}

func (m *fileStore) Options() store.Options {
	return m.options
}

func (m *fileStore) List(opts ...store.ListOption) ([]string, error) {
	listOptions := store.ListOptions{}

	for _, o := range opts {
		o(&listOptions)
	}

	// TODO apply prefix/suffix in range query
	allKeys := m.list(listOptions.Limit, listOptions.Offset)

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

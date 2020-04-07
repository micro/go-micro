// Package local is a file system backed store
package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/micro/go-micro/v2/store"
	micro_store "github.com/micro/go-micro/v2/store"
	bolt "go.etcd.io/bbolt"

	"github.com/pkg/errors"
)

var (
	// DefaultDatabase is the namespace that the bbolt store
	// will use if no namespace is provided.
	DefaultDatabase = "micro"
	// DefaultDir is the default directory for bbolt files
	DefaultDir = os.TempDir()
)

// NewStore returns a memory store
func NewStore(opts ...store.Option) store.Store {
	s := &fileStore{
		options: store.Options{},
	}
	for _, o := range opts {
		o(&s.options)
	}
	return s
}

type fileStore struct {
	options      store.Options
	dir          string
	fileName     string
	fullFilePath string
}

func (m *fileStore) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&m.options)
	}
	if m.options.Database == "" {
		m.options.Database = DefaultDatabase
	}
	if m.options.Table == "" {
		// bbolt requires bucketname to not be empty
		m.options.Table = "default"
	}
	dir := filepath.Join(DefaultDir, "micro")
	fname := m.options.Database + ".db"
	_ = os.Mkdir(dir, 0700)
	m.dir = dir
	m.fileName = fname
	m.fullFilePath = filepath.Join(dir, fname)
	return nil
}

func (m *fileStore) String() string {
	return "local"
}

func (m *fileStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	readOpts := store.ReadOptions{}
	for _, o := range opts {
		o(&readOpts)
	}

	var keys []string

	// Handle Prefix / suffix
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

func (m *fileStore) get(k string) (*store.Record, error) {
	if len(m.options.Table) > 0 {
		k = m.options.Table + "/" + k
	}
	if len(m.options.Database) > 0 {
		k = m.options.Database + "/" + k
	}
	store, err := bolt.Open(m.fullFilePath, 0700, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	defer store.Close()
	err = store.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(m.options.Table))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var value []byte
	store.View(func(tx *bolt.Tx) error {
		// @todo this is still very experimental...
		bucket := tx.Bucket([]byte(m.options.Table))
		value = bucket.Get([]byte(k))
		return nil
	})
	if value == nil {
		return nil, micro_store.ErrNotFound
	}
	storedRecord := &internalRecord{}
	err = json.Unmarshal(value, storedRecord)
	if err != nil {
		return nil, err
	}
	newRecord := &micro_store.Record{}
	newRecord.Key = storedRecord.Key
	newRecord.Value = storedRecord.Value
	if !storedRecord.ExpiresAt.IsZero() {
		if storedRecord.ExpiresAt.Before(time.Now()) {
			return nil, micro_store.ErrNotFound
		}
		newRecord.Expiry = time.Until(storedRecord.ExpiresAt)
	}

	return newRecord, nil
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

func (m *fileStore) set(r *store.Record) error {
	key := r.Key
	if len(m.options.Table) > 0 {
		key = m.options.Table + "/" + key
	}
	if len(m.options.Database) > 0 {
		key = m.options.Database + "/" + key
	}

	// copy the incoming record and then
	// convert the expiry in to a hard timestamp
	i := &internalRecord{}
	i.Key = r.Key
	i.Value = r.Value
	if r.Expiry != 0 {
		i.ExpiresAt = time.Now().Add(r.Expiry)
	}

	iJSON, _ := json.Marshal(i)

	store, err := bolt.Open(m.fullFilePath, 0700, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer store.Close()
	return store.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(m.options.Table))
		if b == nil {
			var err error
			b, err = tx.CreateBucketIfNotExists([]byte(m.options.Table))
			if err != nil {
				return err
			}
		}
		return b.Put([]byte(key), iJSON)
	})
}

func (m *fileStore) Delete(key string, opts ...store.DeleteOption) error {
	deleteOptions := store.DeleteOptions{}
	for _, o := range opts {
		o(&deleteOptions)
	}
	return m.delete(key)
}

func (m *fileStore) delete(key string) error {
	if len(m.options.Table) > 0 {
		key = m.options.Table + "/" + key
	}
	if len(m.options.Database) > 0 {
		key = m.options.Database + "/" + key
	}
	store, err := bolt.Open(m.fullFilePath, 0700, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer store.Close()
	return store.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(m.options.Table))
		if b == nil {
			var err error
			b, err = tx.CreateBucketIfNotExists([]byte(m.options.Table))
			if err != nil {
				return err
			}
		}
		err := b.Delete([]byte(key))
		return err
	})
}

func (m *fileStore) deleteAll() error {
	return os.Remove(m.fullFilePath)
}

func (m *fileStore) Options() store.Options {
	return m.options
}

func (m *fileStore) List(opts ...store.ListOption) ([]string, error) {
	listOptions := store.ListOptions{}

	for _, o := range opts {
		o(&listOptions)
	}
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

func (m *fileStore) list(limit, offset uint) []string {
	allItems := []string{}
	store, err := bolt.Open(m.fullFilePath, 0700, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		fmt.Println("Error creating file:", err)
	}
	defer store.Close()
	store.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(m.options.Table))
		if b == nil {
			var err error
			b, err = tx.CreateBucketIfNotExists([]byte(m.options.Table))
			if err != nil {
				return err
			}
		}
		// @todo very inefficient
		if err := b.ForEach(func(k, v []byte) error {
			storedRecord := &internalRecord{}
			err := json.Unmarshal(v, storedRecord)
			if err != nil {
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
	i := 0
	for _, k := range allItems {
		if len(m.options.Database) > 0 {
			k = strings.TrimPrefix(k, m.options.Database+"/")
		}
		if len(m.options.Table) > 0 {
			k = strings.TrimPrefix(k, m.options.Table+"/")
		}
		allKeys[i] = k
		i++
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

type internalRecord struct {
	Key       string
	Value     []byte
	ExpiresAt time.Time
}

// Package local is a file system backed store
package local

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

// NewStore returns a memory store
func NewStore(opts ...store.Option) store.Store {
	s := &localStore{
		options: store.Options{},
	}
	for _, o := range opts {
		o(&s.options)
	}
	return s
}

type localStore struct {
	options      store.Options
	dir          string
	fileName     string
	fullFilePath string
}

func (m *localStore) Init(opts ...store.Option) error {
	// m.store.Flush()
	for _, o := range opts {
		o(&m.options)
	}
	if m.options.Namespace == "" {
		m.options.Namespace = "default"
	}
	dir := filepath.Join(os.TempDir(), "micro")
	fname := m.options.Namespace + ".db"
	_ = os.Mkdir(dir, 0700)
	m.dir = dir
	m.fileName = fname
	m.fullFilePath = filepath.Join(dir, fname)
	return nil
}

func (m *localStore) String() string {
	return "local"
}

func (m *localStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
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
			return nil, errors.Wrap(err, "Memory: Read couldn't List()")
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

func (m *localStore) get(k string) (*store.Record, error) {
	if len(m.options.Suffix) > 0 {
		k = k + m.options.Suffix
	}
	if len(m.options.Prefix) > 0 {
		k = m.options.Prefix + "/" + k
	}
	if len(m.options.Namespace) > 0 {
		k = m.options.Namespace + "/" + k
	}
	store, err := bolt.Open(m.fullFilePath, 0700, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		fmt.Println("Error creating file:", err)
		return nil, err
	}
	defer store.Close()
	err = store.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(m.options.Namespace))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error creating bucket: ", err)
		return nil, err
	}
	var value []byte
	store.View(func(tx *bolt.Tx) error {
		// @todo this is still very experimental...
		bucket := tx.Bucket([]byte(m.Options().Namespace))
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
		newRecord.Expiry = time.Until(storedRecord.ExpiresAt)
	}

	return newRecord, nil
}

func (m *localStore) Write(r *store.Record, opts ...store.WriteOption) error {
	writeOpts := store.WriteOptions{}
	for _, o := range opts {
		o(&writeOpts)
	}

	if len(opts) > 0 {
		// Copy the record before applying options, or the incoming record will be mutated
		newRecord := store.Record{}
		newRecord.Key = r.Key
		newRecord.Value = make([]byte, len(r.Value))
		copy(newRecord.Value, r.Value)
		newRecord.Expiry = r.Expiry

		if !writeOpts.Expiry.IsZero() {
			newRecord.Expiry = time.Until(writeOpts.Expiry)
		}
		if writeOpts.TTL != 0 {
			newRecord.Expiry = writeOpts.TTL
		}
		m.set(&newRecord)
	} else {
		m.set(r)
	}
	return nil
}

func (m *localStore) set(r *store.Record) {
	key := r.Key
	if len(m.options.Suffix) > 0 {
		key = key + m.options.Suffix
	}
	if len(m.options.Prefix) > 0 {
		key = m.options.Prefix + "/" + key
	}
	if len(m.options.Namespace) > 0 {
		key = m.options.Namespace + "/" + key
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
		fmt.Println("Error creating file:", err)
	}
	defer store.Close()
	store.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(m.options.Namespace))
		fmt.Println("b", b)
		err := b.Put([]byte(key), iJSON)
		return err
	})
}

func (m *localStore) Delete(key string, opts ...store.DeleteOption) error {
	deleteOptions := store.DeleteOptions{}
	for _, o := range opts {
		o(&deleteOptions)
	}
	m.delete(key)
	return nil
}

func (m *localStore) delete(key string) {
	if len(m.options.Suffix) > 0 {
		key = key + m.options.Suffix
	}
	if len(m.options.Prefix) > 0 {
		key = m.options.Prefix + "/" + key
	}
	if len(m.options.Namespace) > 0 {
		key = m.options.Namespace + "/" + key
	}
	store, err := bolt.Open(m.fullFilePath, 0700, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer store.Close()
	store.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(m.options.Namespace))
		err := b.Delete([]byte(key))
		return err
	})
}

func (m *localStore) Options() store.Options {
	return m.options
}

func (m *localStore) List(opts ...store.ListOption) ([]string, error) {
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

func (m *localStore) list(limit, offset uint) []string {
	allItems := []string{}
	store, err := bolt.Open(m.fullFilePath, 0700, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		fmt.Println("Error creating file:", err)
	}
	defer store.Close()
	store.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(m.options.Namespace))
		// Iterate over items in sorted key order.
		if err := b.ForEach(func(k, v []byte) error {
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
		if len(m.options.Suffix) > 0 {
			k = strings.TrimSuffix(k, m.options.Suffix)
		}
		if len(m.options.Namespace) > 0 {
			k = strings.TrimPrefix(k, m.options.Namespace+"/")
		}
		if len(m.options.Prefix) > 0 {
			k = strings.TrimPrefix(k, m.options.Prefix+"/")
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

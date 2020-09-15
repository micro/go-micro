package file

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/micro/go-micro/v3/store"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

// NewBlobStore returns a blob file store
func NewBlobStore() (store.BlobStore, error) {
	// ensure the parent directory exists
	os.MkdirAll(DefaultDir, 0700)

	// open the connection to the database
	dbPath := filepath.Join(DefaultDir, "micro.db")
	db, err := bolt.Open(dbPath, 0700, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, errors.Wrap(err, "Error connecting to database")
	}

	// return the blob store
	return &blobStore{db}, nil
}

type blobStore struct {
	db *bolt.DB
}

func (b *blobStore) Read(key string, opts ...store.BlobOption) (io.Reader, error) {
	// validate the key
	if len(key) == 0 {
		return nil, store.ErrMissingKey
	}

	// parse the options
	var options store.BlobOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.Namespace) == 0 {
		options.Namespace = "micro"
	}

	// execute the transaction
	var value []byte
	readValue := func(tx *bolt.Tx) error {
		// check for the namespaces bucket
		bucket := tx.Bucket([]byte(options.Namespace))
		if bucket == nil {
			return store.ErrNotFound
		}

		// look for the blob within the bucket
		value = bucket.Get([]byte(key))
		if value == nil {
			return store.ErrNotFound
		}

		return nil
	}
	if err := b.db.View(readValue); err != nil {
		return nil, err
	}

	// return the blob
	return bytes.NewBuffer(value), nil
}

func (b *blobStore) Write(key string, blob io.Reader, opts ...store.BlobOption) error {
	// validate the key
	if len(key) == 0 {
		return store.ErrMissingKey
	}

	// parse the options
	var options store.BlobOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.Namespace) == 0 {
		options.Namespace = "micro"
	}

	// execute the transaction
	return b.db.Update(func(tx *bolt.Tx) error {
		// create the bucket
		bucket, err := tx.CreateBucketIfNotExists([]byte(options.Namespace))
		if err != nil {
			return err
		}

		// write to the bucket
		value, err := ioutil.ReadAll(blob)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(key), value)
	})
}

func (b *blobStore) Delete(key string, opts ...store.BlobOption) error {
	// validate the key
	if len(key) == 0 {
		return store.ErrMissingKey
	}

	// parse the options
	var options store.BlobOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.Namespace) == 0 {
		options.Namespace = "micro"
	}

	// execute the transaction
	return b.db.Update(func(tx *bolt.Tx) error {
		// check for the namespaces bucket
		bucket := tx.Bucket([]byte(options.Namespace))
		if bucket == nil {
			return store.ErrNotFound
		}

		if bucket.Get([]byte(key)) == nil {
			return store.ErrNotFound
		}

		return bucket.Delete([]byte(key))
	})
}

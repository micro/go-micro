package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/micro/go-micro/v3/store"
	"github.com/micro/go-micro/v3/store/file"
	"github.com/stretchr/testify/assert"
)

func cleanup(db string, s store.Store) {
	s.Close()
	dir := filepath.Join(file.DefaultDir, db+"/")
	os.RemoveAll(dir)
}

func TestRead(t *testing.T) {
	cf := NewStore(file.NewStore())
	cf.Init()
	cfInt := cf.(*cache)
	defer cleanup(file.DefaultDatabase, cf)

	_, err := cf.Read("key1")
	assert.Error(t, err, "Unexpected record")
	cfInt.b.Write(&store.Record{
		Key:   "key1",
		Value: []byte("foo"),
	})
	recs, err := cf.Read("key1")
	assert.NoError(t, err)
	assert.Len(t, recs, 1, "Expected a record to be pulled from file store")
	recs, err = cfInt.m.Read("key1")
	assert.NoError(t, err)
	assert.Len(t, recs, 1, "Expected a memory store to be populatedfrom file store")

}

func TestWrite(t *testing.T) {
	cf := NewStore(file.NewStore())
	cf.Init()
	cfInt := cf.(*cache)
	defer cleanup(file.DefaultDatabase, cf)

	cf.Write(&store.Record{
		Key:   "key1",
		Value: []byte("foo"),
	})
	recs, _ := cfInt.m.Read("key1")
	assert.Len(t, recs, 1, "Expected a record in the memory store")
	recs, _ = cfInt.b.Read("key1")
	assert.Len(t, recs, 1, "Expected a record in the file store")

}

func TestDelete(t *testing.T) {
	cf := NewStore(file.NewStore())
	cf.Init()
	cfInt := cf.(*cache)
	defer cleanup(file.DefaultDatabase, cf)

	cf.Write(&store.Record{
		Key:   "key1",
		Value: []byte("foo"),
	})
	recs, _ := cfInt.m.Read("key1")
	assert.Len(t, recs, 1, "Expected a record in the memory store")
	recs, _ = cfInt.b.Read("key1")
	assert.Len(t, recs, 1, "Expected a record in the file store")
	cf.Delete("key1")

	_, err := cfInt.m.Read("key1")
	assert.Error(t, err, "Expected no records in memory store")
	_, err = cfInt.b.Read("key1")
	assert.Error(t, err, "Expected no records in file store")

}

func TestList(t *testing.T) {
	cf := NewStore(file.NewStore())
	cf.Init()
	cfInt := cf.(*cache)
	defer cleanup(file.DefaultDatabase, cf)

	keys, err := cf.List()
	assert.NoError(t, err)
	assert.Len(t, keys, 0)
	cfInt.b.Write(&store.Record{
		Key:   "key1",
		Value: []byte("foo"),
	})

	cfInt.b.Write(&store.Record{
		Key:   "key2",
		Value: []byte("foo"),
	})
	keys, err = cf.List()
	assert.NoError(t, err)
	assert.Len(t, keys, 2)

}

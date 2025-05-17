//go:build integration
// +build integration

package pgx

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-micro.dev/v5/store"
)

type testObj struct {
	One string
	Two int64
}

func TestPostgres(t *testing.T) {
	t.Run("ReadWrite", func(t *testing.T) {
		s := NewStore(store.Nodes("postgresql://postgres@localhost:5432/?sslmode=disable"))
		b, _ := json.Marshal(testObj{
			One: "1",
			Two: 2,
		})
		err := s.Write(&store.Record{
			Key:   "foobar/baz",
			Value: b,
			Metadata: map[string]interface{}{
				"meta1": "val1",
			},
		})
		assert.NoError(t, err)
		recs, err := s.Read("foobar/baz")
		assert.NoError(t, err)
		assert.Len(t, recs, 1)
		assert.Equal(t, "foobar/baz", recs[0].Key)
		assert.Len(t, recs[0].Metadata, 1)
		assert.Equal(t, "val1", recs[0].Metadata["meta1"])

		var tobj testObj
		assert.NoError(t, json.Unmarshal(recs[0].Value, &tobj))
		assert.Equal(t, "1", tobj.One)
		assert.Equal(t, int64(2), tobj.Two)
	})
	t.Run("Prefix", func(t *testing.T) {
		s := NewStore(store.Nodes("postgresql://postgres@localhost:5432/?sslmode=disable"))
		b, _ := json.Marshal(testObj{
			One: "1",
			Two: 2,
		})
		err := s.Write(&store.Record{
			Key:   "foo/bar",
			Value: b,
			Metadata: map[string]interface{}{
				"meta1": "val1",
			},
		})
		assert.NoError(t, err)
		err = s.Write(&store.Record{
			Key:   "foo/baz",
			Value: b,
			Metadata: map[string]interface{}{
				"meta1": "val1",
			},
		})
		assert.NoError(t, err)
		recs, err := s.Read("foo/", store.ReadPrefix())
		assert.NoError(t, err)
		assert.Len(t, recs, 2)
		assert.Equal(t, "foo/bar", recs[0].Key)
		assert.Equal(t, "foo/baz", recs[1].Key)
	})

	t.Run("MultipleTables", func(t *testing.T) {
		s1 := NewStore(store.Nodes("postgresql://postgres@localhost:5432/?sslmode=disable"), store.Table("t1"))
		s2 := NewStore(store.Nodes("postgresql://postgres@localhost:5432/?sslmode=disable"), store.Table("t2"))
		b1, _ := json.Marshal(testObj{
			One: "1",
			Two: 2,
		})
		err := s1.Write(&store.Record{
			Key:   "foo/bar",
			Value: b1,
		})
		assert.NoError(t, err)
		b2, _ := json.Marshal(testObj{
			One: "1",
			Two: 2,
		})
		err = s2.Write(&store.Record{
			Key:   "foo/baz",
			Value: b2,
		})
		assert.NoError(t, err)
		recs1, err := s1.List()
		assert.NoError(t, err)
		assert.Len(t, recs1, 1)
		assert.Equal(t, "foo/bar", recs1[0])

		recs2, err := s2.List()
		assert.NoError(t, err)
		assert.Len(t, recs2, 1)
		assert.Equal(t, "foo/baz", recs2[0])
	})

	t.Run("MultipleDBs", func(t *testing.T) {
		s1 := NewStore(store.Nodes("postgresql://postgres@localhost:5432/?sslmode=disable"), store.Database("d1"))
		s2 := NewStore(store.Nodes("postgresql://postgres@localhost:5432/?sslmode=disable"), store.Database("d2"))
		b1, _ := json.Marshal(testObj{
			One: "1",
			Two: 2,
		})
		err := s1.Write(&store.Record{
			Key:   "foo/bar",
			Value: b1,
		})
		assert.NoError(t, err)
		b2, _ := json.Marshal(testObj{
			One: "1",
			Two: 2,
		})
		err = s2.Write(&store.Record{
			Key:   "foo/baz",
			Value: b2,
		})
		assert.NoError(t, err)
		recs1, err := s1.List()
		assert.NoError(t, err)
		assert.Len(t, recs1, 1)
		assert.Equal(t, "foo/bar", recs1[0])

		recs2, err := s2.List()
		assert.NoError(t, err)
		assert.Len(t, recs2, 1)
		assert.Equal(t, "foo/baz", recs2[0])
	})
}

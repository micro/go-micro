// Package test provides a way to run tests against all the various implementations of the Store interface.
// It can't live in the store package itself because of circular import issues
package test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/kr/pretty"
	"github.com/micro/go-micro/v3/store"
	"github.com/micro/go-micro/v3/store/cache"
	"github.com/micro/go-micro/v3/store/file"
	"github.com/micro/go-micro/v3/store/memory"
)

func fileStoreCleanup(db string, s store.Store) {
	s.Close()
	dir := filepath.Join(file.DefaultDir, db+"/")
	os.RemoveAll(dir)
}

func cockroachStoreCleanup(db string, s store.Store) {
	keys, _ := s.List()
	for _, k := range keys {
		s.Delete(k)
	}
	s.Close()
}

func memoryCleanup(db string, s store.Store) {
	s.Close()
}

func cacheCleanup(db string, s store.Store) {
	s.Close()
}

func TestStoreReInit(t *testing.T) {
	tcs := []struct {
		name    string
		s       store.Store
		cleanup func(db string, s store.Store)
	}{
		{name: "file", s: file.NewStore(store.Table("aaa")), cleanup: fileStoreCleanup},
		{name: "memory", s: memory.NewStore(store.Table("aaa")), cleanup: memoryCleanup},
		{name: "cache", s: cache.NewStore(memory.NewStore(store.Table("aaa"))), cleanup: cacheCleanup},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			defer tc.cleanup(file.DefaultDatabase, tc.s)
			tc.s.Init(store.Table("bbb"))
			if tc.s.Options().Table != "bbb" {
				t.Error("Init didn't reinitialise the store")
			}
		})
	}
}

func TestStoreBasic(t *testing.T) {
	tcs := []struct {
		name    string
		s       store.Store
		cleanup func(db string, s store.Store)
	}{
		{name: "file", s: file.NewStore(), cleanup: fileStoreCleanup},
		{name: "memory", s: memory.NewStore(), cleanup: memoryCleanup},
		{name: "cache", s: cache.NewStore(memory.NewStore()), cleanup: cacheCleanup},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			defer tc.cleanup(file.DefaultDatabase, tc.s)
			runStoreTest(tc.s, t)
		})
	}

}

func TestStoreTable(t *testing.T) {
	tcs := []struct {
		name    string
		s       store.Store
		cleanup func(db string, s store.Store)
	}{
		{name: "file", s: file.NewStore(store.Table("testTable")), cleanup: fileStoreCleanup},
		{name: "memory", s: memory.NewStore(store.Table("testTable")), cleanup: memoryCleanup},
		{name: "cache", s: cache.NewStore(memory.NewStore(store.Table("testTable"))), cleanup: cacheCleanup},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			defer tc.cleanup(file.DefaultDatabase, tc.s)
			runStoreTest(tc.s, t)
		})
	}
}

func TestStoreDatabase(t *testing.T) {
	tcs := []struct {
		name    string
		s       store.Store
		cleanup func(db string, s store.Store)
	}{
		{name: "file", s: file.NewStore(store.Database("testdb")), cleanup: fileStoreCleanup},
		{name: "memory", s: memory.NewStore(store.Database("testdb")), cleanup: memoryCleanup},
		{name: "cache", s: cache.NewStore(memory.NewStore(store.Database("testdb"))), cleanup: cacheCleanup},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			defer tc.cleanup("testdb", tc.s)
			runStoreTest(tc.s, t)
		})
	}
}

func TestStoreDatabaseTable(t *testing.T) {
	tcs := []struct {
		name    string
		s       store.Store
		cleanup func(db string, s store.Store)
	}{
		{name: "file", s: file.NewStore(store.Database("testdb"), store.Table("testTable")), cleanup: fileStoreCleanup},
		{name: "memory", s: memory.NewStore(store.Database("testdb"), store.Table("testTable")), cleanup: memoryCleanup},
		{name: "cache", s: cache.NewStore(memory.NewStore(store.Database("testdb"), store.Table("testTable"))), cleanup: cacheCleanup},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			defer tc.cleanup("testdb", tc.s)
			runStoreTest(tc.s, t)
		})
	}
}

func runStoreTest(s store.Store, t *testing.T) {
	if len(os.Getenv("IN_TRAVIS_CI")) == 0 {
		t.Logf("Options %s %v\n", s.String(), s.Options())
	}

	expiryTests(s, t)
	suffixPrefixExpiryTests(s, t)
	readTests(s, t)
	listTests(s, t)

}

func readTests(s store.Store, t *testing.T) {
	// Test Table, Suffix and WriteOptions
	if err := s.Write(&store.Record{
		Key:    "foofoobarbar",
		Value:  []byte("something"),
		Expiry: time.Millisecond * 100,
	}); err != nil {
		t.Error(err)
	}
	if err := s.Write(&store.Record{
		Key:    "foofoo",
		Value:  []byte("something"),
		Expiry: time.Millisecond * 100,
	}); err != nil {
		t.Error(err)
	}
	if err := s.Write(&store.Record{
		Key:    "barbar",
		Value:  []byte("something"),
		Expiry: time.Millisecond * 100,
	}); err != nil {
		t.Error(err)
	}

	if results, err := s.Read("foo", store.ReadPrefix(), store.ReadSuffix()); err != nil {
		t.Error(err)
	} else {
		if len(results) != 1 {
			t.Errorf("Expected 1 results, got %d: %# v", len(results), spew.Sdump(results))
		}
	}

	time.Sleep(time.Millisecond * 100)

	if results, err := s.List(); err != nil {
		t.Fatalf("List failed: %s", err)
	} else {
		if len(results) != 0 {
			t.Fatalf("Expiry options were not effective, results :%v", spew.Sdump(results))
		}
	}

	// write the following records
	for i := 0; i < 10; i++ {
		s.Write(&store.Record{
			Key:   fmt.Sprintf("a%d", i),
			Value: []byte{},
		})
	}

	// read back a few records
	if results, err := s.Read("a", store.ReadLimit(5), store.ReadPrefix()); err != nil {
		t.Error(err)
	} else {
		if len(results) != 5 {
			t.Fatal("Expected 5 results, got ", len(results))
		}
		if !strings.HasPrefix(results[0].Key, "a") {
			t.Fatalf("Expected a prefix, got %s", results[0].Key)
		}
	}

	// read the rest back
	if results, err := s.Read("a", store.ReadLimit(30), store.ReadOffset(5), store.ReadPrefix()); err != nil {
		t.Fatal(err)
	} else {
		if len(results) != 5 {
			t.Fatal("Expected 5 results, got ", len(results))
		}
	}
}

func listTests(s store.Store, t *testing.T) {
	for i := 0; i < 10; i++ {
		s.Write(&store.Record{Key: fmt.Sprintf("List%d", i), Value: []byte("bar")})
	}

	recs, err := s.List(store.ListPrefix("List"))
	if err != nil {
		t.Fatalf("Error listing records %s", err)
	}
	if len(recs) != 10 {
		t.Fatalf("Expected 10 records, received %d", len(recs))
	}

	recs, err = s.List(store.ListPrefix("List"), store.ListLimit(5))
	if err != nil {
		t.Fatalf("Error listing records %s", err)
	}
	if len(recs) != 5 {
		t.Fatalf("Expected 5 records, received %d", len(recs))
	}

	recs, err = s.List(store.ListPrefix("List"), store.ListOffset(6))
	if err != nil {
		t.Fatalf("Error listing records %s", err)
	}
	if len(recs) != 4 {
		t.Fatalf("Expected 4 records, received %d %+v", len(recs), recs)
	}

	recs, err = s.List(store.ListPrefix("List"), store.ListOffset(6), store.ListLimit(2))
	if err != nil {
		t.Fatalf("Error listing records %s", err)
	}
	if len(recs) != 2 {
		t.Fatalf("Expected 2 records, received %d %+v", len(recs), recs)
	}

	for i := 0; i < 10; i++ {
		s.Write(&store.Record{Key: fmt.Sprintf("ListOffset%d", i), Value: []byte("bar")})
	}

	recs, err = s.List(store.ListPrefix("ListOffset"), store.ListOffset(6))
	if err != nil {
		t.Fatalf("Error listing records %s", err)
	}
	if len(recs) != 4 {
		t.Fatalf("Expected 4 records, received %d %+v", len(recs), recs)
	}

}

func expiryTests(s store.Store, t *testing.T) {
	// Read and Write an expiring Record
	if err := s.Write(&store.Record{
		Key:    "Hello",
		Value:  []byte("World"),
		Expiry: time.Millisecond * 150,
	}); err != nil {
		t.Error(err)
	}

	if r, err := s.Read("Hello"); err != nil {
		t.Fatal(err)
	} else {
		if len(r) != 1 {
			t.Error("Read returned multiple records")
		}
		if r[0].Key != "Hello" {
			t.Errorf("Expected %s, got %s", "Hello", r[0].Key)
		}
		if string(r[0].Value) != "World" {
			t.Errorf("Expected %s, got %s", "World", r[0].Value)
		}
	}

	// wait for expiry
	time.Sleep(time.Millisecond * 200)

	if _, err := s.Read("Hello"); err != store.ErrNotFound {
		t.Errorf("Expected %# v, got %# v", store.ErrNotFound, err)
	}

	// exercise the records expiry
	s.Write(&store.Record{Key: "aaa", Value: []byte("bbb"), Expiry: 1 * time.Second})
	s.Write(&store.Record{Key: "aaaa", Value: []byte("bbb"), Expiry: 1 * time.Second})
	s.Write(&store.Record{Key: "aaaaa", Value: []byte("bbb"), Expiry: 1 * time.Second})
	results, err := s.Read("a", store.ReadPrefix())
	if err != nil {
		t.Error(err)
	}
	if len(results) != 3 {
		t.Fatalf("Results should have returned 3 records, returned %d", len(results))
	}
	time.Sleep(1 * time.Second)
	results, err = s.Read("a", store.ReadPrefix())
	if err != nil {
		t.Error(err)
	}
	if len(results) != 0 {
		t.Fatal("Results should have returned 0 records")
	}
}

func suffixPrefixExpiryTests(s store.Store, t *testing.T) {
	// Write 3 records with various expiry and get with Prefix
	records := []*store.Record{
		&store.Record{
			Key:   "foo",
			Value: []byte("foofoo"),
		},
		&store.Record{
			Key:    "foobar",
			Value:  []byte("foobarfoobar"),
			Expiry: 1 * time.Second,
		},
	}

	for _, r := range records {
		if err := s.Write(r); err != nil {
			t.Errorf("Couldn't write k: %s, v: %# v (%s)", r.Key, pretty.Formatter(r.Value), err)
		}
	}

	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %#v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 2 {
			t.Errorf("Expected 2 items, got %d", len(results))
		}
	}

	// wait for the expiry
	time.Sleep(1 * time.Second)

	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else if len(results) != 1 {
		t.Errorf("Expected 1 item, got %d", len(results))
	}

	if err := s.Delete("foo"); err != nil {
		t.Errorf("Delete failed (%v)", err)
	}

	if results, err := s.Read("foo"); err != store.ErrNotFound {
		t.Errorf("Expected read failure read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 0 {
			t.Errorf("Expected 0 items, got %d (%# v)", len(results), spew.Sdump(results))
		}
	}

	// Write 3 records with various expiry and get with Suffix
	records = []*store.Record{
		&store.Record{
			Key:   "foo",
			Value: []byte("foofoo"),
		},
		&store.Record{
			Key:   "barfoo",
			Value: []byte("barfoobarfoo"),

			Expiry: time.Second * 1,
		},
		&store.Record{
			Key:    "bazbarfoo",
			Value:  []byte("bazbarfoobazbarfoo"),
			Expiry: 2 * time.Second,
		},
	}
	for _, r := range records {
		if err := s.Write(r); err != nil {
			t.Errorf("Couldn't write k: %s, v: %# v (%s)", r.Key, pretty.Formatter(r.Value), err)
		}
	}
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 3 {
			t.Errorf("Expected 3 items, got %d", len(results))
			//t.Logf("Table test: %v\n", spew.Sdump(results))
		}

	}
	time.Sleep(time.Second * 1)
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 2 {
			t.Errorf("Expected 2 items, got %d", len(results))
			//t.Logf("Table test: %v\n", spew.Sdump(results))
		}

	}
	time.Sleep(time.Second * 1)
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 1 {
			t.Errorf("Expected 1 item, got %d", len(results))
			//	t.Logf("Table test: %# v\n", spew.Sdump(results))
		}
	}
	if err := s.Delete("foo"); err != nil {
		t.Errorf("Delete failed (%v)", err)
	}
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 0 {
			t.Errorf("Expected 0 items, got %d (%# v)", len(results), spew.Sdump(results))
		}
	}
}

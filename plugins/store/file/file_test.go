package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/kr/pretty"
	"github.com/micro/go-micro/v2/store"
)

func cleanup(db string, s store.Store) {
	s.Close()
	dir := filepath.Join(DefaultDir, db+"/")
	os.RemoveAll(dir)
}

func TestFileStoreReInit(t *testing.T) {
	s := NewStore(store.Table("aaa"))
	defer cleanup(DefaultDatabase, s)
	s.Init(store.Table("bbb"))
	if s.Options().Table != "bbb" {
		t.Error("Init didn't reinitialise the store")
	}
}

func TestFileStoreBasic(t *testing.T) {
	s := NewStore()
	defer cleanup(DefaultDatabase, s)
	fileTest(s, t)
}

func TestFileStoreTable(t *testing.T) {
	s := NewStore(store.Table("testTable"))
	defer cleanup(DefaultDatabase, s)
	fileTest(s, t)
}

func TestFileStoreDatabase(t *testing.T) {
	s := NewStore(store.Database("testdb"))
	defer cleanup("testdb", s)
	fileTest(s, t)
}

func TestFileStoreDatabaseTable(t *testing.T) {
	s := NewStore(store.Table("testTable"), store.Database("testdb"))
	defer cleanup("testdb", s)
	fileTest(s, t)
}

func fileTest(s store.Store, t *testing.T) {
	if len(os.Getenv("IN_TRAVIS_CI")) == 0 {
		t.Logf("Options %s %v\n", s.String(), s.Options())
	}
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

	// Write 3 records with various expiry and get with Table
	records := []*store.Record{
		&store.Record{
			Key:   "foo",
			Value: []byte("foofoo"),
		},
		&store.Record{
			Key:    "foobar",
			Value:  []byte("foobarfoobar"),
			Expiry: time.Millisecond * 100,
		},
	}

	for _, r := range records {
		if err := s.Write(r); err != nil {
			t.Errorf("Couldn't write k: %s, v: %# v (%s)", r.Key, pretty.Formatter(r.Value), err)
		}
	}

	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 2 {
			t.Errorf("Expected 2 items, got %d", len(results))
			//t.Logf("Table test: %v\n", spew.Sdump(results))
		}
	}

	// wait for the expiry
	time.Sleep(time.Millisecond * 200)

	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else if len(results) != 1 {
		t.Errorf("Expected 1 item, got %d", len(results))
		//t.Logf("Table test: %v\n", spew.Sdump(results))
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

			Expiry: time.Millisecond * 100,
		},
		&store.Record{
			Key:    "bazbarfoo",
			Value:  []byte("bazbarfoobazbarfoo"),
			Expiry: 2 * time.Millisecond * 100,
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
	time.Sleep(time.Millisecond * 100)
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 2 {
			t.Errorf("Expected 2 items, got %d", len(results))
			//t.Logf("Table test: %v\n", spew.Sdump(results))
		}

	}
	time.Sleep(time.Millisecond * 100)
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

	// Test Table, Suffix and WriteOptions
	if err := s.Write(&store.Record{
		Key:   "foofoobarbar",
		Value: []byte("something"),
	}, store.WriteTTL(time.Millisecond*100)); err != nil {
		t.Error(err)
	}
	if err := s.Write(&store.Record{
		Key:   "foofoo",
		Value: []byte("something"),
	}, store.WriteExpiry(time.Now().Add(time.Millisecond*100))); err != nil {
		t.Error(err)
	}
	if err := s.Write(&store.Record{
		Key:   "barbar",
		Value: []byte("something"),
		// TTL has higher precedence than expiry
	}, store.WriteExpiry(time.Now().Add(time.Hour)), store.WriteTTL(time.Millisecond*100)); err != nil {
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
		t.Errorf("List failed: %s", err)
	} else {
		if len(results) != 0 {
			t.Errorf("Expiry options were not effective, results :%v", spew.Sdump(results))
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

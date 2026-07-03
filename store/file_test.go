package store

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/kr/pretty"
)

func newTestFileStore(t *testing.T, opts ...Option) Store {
	t.Helper()
	opts = append(opts, DirOption(t.TempDir()))
	s := NewStore(opts...)
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("failed to close file store: %v", err)
		}
	})
	return s
}

func TestFileStoreReInit(t *testing.T) {
	s := newTestFileStore(t, Table("aaa"))
	s.Init(Table("bbb"))
	if s.Options().Table != "bbb" {
		t.Error("Init didn't reinitialise the store")
	}
}

func TestFileStoreBasic(t *testing.T) {
	s := newTestFileStore(t)
	fileTest(s, t)
}

func TestFileStoreTable(t *testing.T) {
	s := newTestFileStore(t, Table("testTable"))
	fileTest(s, t)
}

func TestFileStoreDatabase(t *testing.T) {
	s := newTestFileStore(t, Database("testdb"))
	fileTest(s, t)
}

func TestFileStoreDatabaseTable(t *testing.T) {
	s := newTestFileStore(t, Table("testTable"), Database("testdb"))
	fileTest(s, t)
}

func fileTest(s Store, t *testing.T) {
	if len(os.Getenv("IN_TRAVIS_CI")) == 0 {
		t.Logf("Options %s %v\n", s.String(), s.Options())
	}
	// Read and Write an expiring Record
	if err := s.Write(&Record{
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

	if _, err := s.Read("Hello"); err != ErrNotFound {
		t.Errorf("Expected %# v, got %# v", ErrNotFound, err)
	}

	// Write 3 records with various expiry and get with Table
	records := []*Record{
		{
			Key:   "foo",
			Value: []byte("foofoo"),
		},
		{
			Key:    "foobar",
			Value:  []byte("foobarfoobar"),
			Expiry: time.Second, // wide window: CI I/O under -race can exceed a 100ms expiry before the read below
		},
	}

	for _, r := range records {
		if err := s.Write(r); err != nil {
			t.Errorf("Couldn't write k: %s, v: %# v (%s)", r.Key, pretty.Formatter(r.Value), err)
		}
	}

	if results, err := s.Read("foo", ReadPrefix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 2 {
			t.Errorf("Expected 2 items, got %d", len(results))
			// t.Logf("Table test: %v\n", spew.Sdump(results))
		}
	}

	// wait for the expiry (must exceed the 1s Expiry above, with margin for slow CI)
	time.Sleep(time.Second * 2)

	if results, err := s.Read("foo", ReadPrefix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else if len(results) != 1 {
		t.Errorf("Expected 1 item, got %d", len(results))
		// t.Logf("Table test: %v\n", spew.Sdump(results))
	}

	if err := s.Delete("foo"); err != nil {
		t.Errorf("Delete failed (%v)", err)
	}

	if results, err := s.Read("foo"); err != ErrNotFound {
		t.Errorf("Expected read failure read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else {
		if len(results) != 0 {
			t.Errorf("Expected 0 items, got %d (%# v)", len(results), spew.Sdump(results))
		}
	}

	// Write records with suffix matches and an already-expired record. Avoid
	// wall-clock boundary sleeps here: under -race/-cover, sleeping exactly the
	// TTL made this assertion flaky on slower CI runners.
	records = []*Record{
		{
			Key:   "foo",
			Value: []byte("foofoo"),
		},
		{
			Key:    "barfoo",
			Value:  []byte("barfoobarfoo"),
			Expiry: -time.Second,
		},
		{
			Key:   "bazbarfoo",
			Value: []byte("bazbarfoobazbarfoo"),
		},
	}
	for _, r := range records {
		if err := s.Write(r); err != nil {
			t.Errorf("Couldn't write k: %s, v: %# v (%s)", r.Key, pretty.Formatter(r.Value), err)
		}
	}
	if results, err := s.Read("foo", ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else if len(results) != 2 {
		t.Errorf("Expected 2 unexpired suffix items, got %d (%# v)", len(results), spew.Sdump(results))
	}
	if err := s.Delete("bazbarfoo"); err != nil {
		t.Errorf("Delete failed (%v)", err)
	}
	if results, err := s.Read("foo", ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else if len(results) != 1 {
		t.Errorf("Expected 1 unexpired suffix item, got %d (%# v)", len(results), spew.Sdump(results))
	}
	if err := s.Delete("foo"); err != nil {
		t.Errorf("Delete failed (%v)", err)
	}
	if results, err := s.Read("foo", ReadSuffix()); err != nil {
		t.Errorf("Couldn't read all \"foo\" keys, got %# v (%s)", spew.Sdump(results), err)
	} else if len(results) != 0 {
		t.Errorf("Expected 0 items, got %d (%# v)", len(results), spew.Sdump(results))
	}

	// Test Table, Suffix and WriteOptions
	if err := s.Write(&Record{
		Key:   "foofoobarbar",
		Value: []byte("something"),
	}, WriteTTL(time.Second)); err != nil {
		t.Error(err)
	}
	if err := s.Write(&Record{
		Key:   "foofoo",
		Value: []byte("something"),
	}, WriteExpiry(time.Now().Add(time.Second))); err != nil {
		t.Error(err)
	}
	if err := s.Write(&Record{
		Key:   "barbar",
		Value: []byte("something"),
		// TTL has higher precedence than expiry
	}, WriteExpiry(time.Now().Add(time.Hour)), WriteTTL(time.Second)); err != nil {
		t.Error(err)
	}

	if results, err := s.Read("foo", ReadPrefix(), ReadSuffix()); err != nil {
		t.Error(err)
	} else {
		if len(results) != 1 {
			t.Errorf("Expected 1 results, got %d: %# v", len(results), spew.Sdump(results))
		}
	}

	time.Sleep(time.Second * 2) // exceed the 1s TTL/expiry above so everything has expired

	if results, err := s.List(); err != nil {
		t.Errorf("List failed: %s", err)
	} else {
		if len(results) != 0 {
			t.Errorf("Expiry options were not effective, results :%v", spew.Sdump(results))
		}
	}

	// write the following records
	for i := 0; i < 10; i++ {
		s.Write(&Record{
			Key:   fmt.Sprintf("a%d", i),
			Value: []byte{},
		})
	}

	// read back a few records
	if results, err := s.Read("a", ReadLimit(5), ReadPrefix()); err != nil {
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
	if results, err := s.Read("a", ReadLimit(30), ReadOffset(5), ReadPrefix()); err != nil {
		t.Fatal(err)
	} else {
		if len(results) != 5 {
			t.Fatal("Expected 5 results, got ", len(results))
		}
	}
}

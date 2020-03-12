package etcd

import (
	"fmt"
	"testing"
	"time"

	"github.com/kr/pretty"
	"github.com/micro/go-micro/v2/store"
)

func TestEtcd(t *testing.T) {
	e := NewStore()
	if err := e.Init(); err != nil {
		t.Fatal(err)
	}
	//basictest(e, t)
}

func basictest(s store.Store, t *testing.T) {
	t.Logf("Testing store %s, with options %# v\n", s.String(), pretty.Formatter(s.Options()))
	// Read and Write an expiring Record
	if err := s.Write(&store.Record{
		Key:    "Hello",
		Value:  []byte("World"),
		Expiry: time.Second * 5,
	}); err != nil {
		t.Fatal(err)
	}
	if r, err := s.Read("Hello"); err != nil {
		t.Fatal(err)
	} else {
		if len(r) != 1 {
			t.Fatal("Read returned multiple records")
		}
		if r[0].Key != "Hello" {
			t.Fatalf("Expected %s, got %s", "Hello", r[0].Key)
		}
		if string(r[0].Value) != "World" {
			t.Fatalf("Expected %s, got %s", "World", r[0].Value)
		}
	}
	time.Sleep(time.Second * 6)
	if records, err := s.Read("Hello"); err != store.ErrNotFound {
		t.Fatalf("Expected %# v, got %# v\nResults were %# v", store.ErrNotFound, err, pretty.Formatter(records))
	}

	// Write 3 records with various expiry and get with prefix
	records := []*store.Record{
		&store.Record{
			Key:   "foo",
			Value: []byte("foofoo"),
		},
		&store.Record{
			Key:    "foobar",
			Value:  []byte("foobarfoobar"),
			Expiry: time.Second * 5,
		},
		&store.Record{
			Key:    "foobarbaz",
			Value:  []byte("foobarbazfoobarbaz"),
			Expiry: 2 * time.Second * 5,
		},
	}
	for _, r := range records {
		if err := s.Write(r); err != nil {
			t.Fatalf("Couldn't write k: %s, v: %# v (%s)", r.Key, pretty.Formatter(r.Value), err)
		}
	}
	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Fatalf("Couldn't read all \"foo\" keys, got %# v (%s)", pretty.Formatter(results), err)
	} else {
		if len(results) != 3 {
			t.Fatalf("Expected 3 items, got %d", len(results))
		}
	}
	time.Sleep(time.Second * 6)
	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Fatalf("Couldn't read all \"foo\" keys, got %# v (%s)", pretty.Formatter(results), err)
	} else {
		if len(results) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(results))
		}
	}
	time.Sleep(time.Second * 5)
	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Fatalf("Couldn't read all \"foo\" keys, got %# v (%s)", pretty.Formatter(results), err)
	} else {
		if len(results) != 1 {
			t.Fatalf("Expected 1 item, got %d", len(results))
		}
	}
	if err := s.Delete("foo", func(d *store.DeleteOptions) {}); err != nil {
		t.Fatalf("Delete failed (%v)", err)
	}
	if results, err := s.Read("foo", store.ReadPrefix()); err != nil {
		t.Fatalf("Couldn't read all \"foo\" keys, got %# v (%s)", pretty.Formatter(results), err)
	} else {
		if len(results) != 0 {
			t.Fatalf("Expected 0 items, got %d (%# v)", len(results), pretty.Formatter(results))
		}
	}

	// Write 3 records with various expiry and get with Suffix
	records = []*store.Record{
		&store.Record{
			Key:   "foo",
			Value: []byte("foofoo"),
		},
		&store.Record{
			Key:    "barfoo",
			Value:  []byte("barfoobarfoo"),
			Expiry: time.Second * 5,
		},
		&store.Record{
			Key:    "bazbarfoo",
			Value:  []byte("bazbarfoobazbarfoo"),
			Expiry: 2 * time.Second * 5,
		},
	}
	for _, r := range records {
		if err := s.Write(r); err != nil {
			t.Fatalf("Couldn't write k: %s, v: %# v (%s)", r.Key, pretty.Formatter(r.Value), err)
		}
	}
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Fatalf("Couldn't read all \"foo\" keys, got %# v (%s)", pretty.Formatter(results), err)
	} else {
		if len(results) != 3 {
			t.Fatalf("Expected 3 items, got %d", len(results))
		}
	}
	time.Sleep(time.Second * 6)
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Fatalf("Couldn't read all \"foo\" keys, got %# v (%s)", pretty.Formatter(results), err)
	} else {
		if len(results) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(results))
		}
		t.Logf("Prefix test: %v\n", pretty.Formatter(results))
	}
	time.Sleep(time.Second * 5)
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Fatalf("Couldn't read all \"foo\" keys, got %# v (%s)", pretty.Formatter(results), err)
	} else {
		if len(results) != 1 {
			t.Fatalf("Expected 1 item, got %d", len(results))
		}
		t.Logf("Prefix test: %# v\n", pretty.Formatter(results))
	}
	if err := s.Delete("foo"); err != nil {
		t.Fatalf("Delete failed (%v)", err)
	}
	if results, err := s.Read("foo", store.ReadSuffix()); err != nil {
		t.Fatalf("Couldn't read all \"foo\" keys, got %# v (%s)", pretty.Formatter(results), err)
	} else {
		if len(results) != 0 {
			t.Fatalf("Expected 0 items, got %d (%# v)", len(results), pretty.Formatter(results))
		}
	}

	// Test Prefix, Suffix and WriteOptions
	if err := s.Write(&store.Record{
		Key:   "foofoobarbar",
		Value: []byte("something"),
	}, store.WriteTTL(time.Millisecond*100)); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(&store.Record{
		Key:   "foofoo",
		Value: []byte("something"),
	}, store.WriteExpiry(time.Now().Add(time.Millisecond*100))); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(&store.Record{
		Key:   "barbar",
		Value: []byte("something"),
		// TTL has higher precedence than expiry
	}, store.WriteExpiry(time.Now().Add(time.Hour)), store.WriteTTL(time.Millisecond*100)); err != nil {
		t.Fatal(err)
	}
	if results, err := s.Read("foo", store.ReadPrefix(), store.ReadSuffix()); err != nil {
		t.Fatal(err)
	} else {
		if len(results) != 1 {
			t.Fatalf("Expected 1 results, got %d: %# v", len(results), pretty.Formatter(results))
		}
	}
	time.Sleep(time.Second * 6)
	if results, err := s.List(); err != nil {
		t.Fatalf("List failed: %s", err)
	} else {
		if len(results) != 0 {
			t.Fatal("Expiry options were not effective")
		}
	}

	s.Init()
	for i := 0; i < 10; i++ {
		s.Write(&store.Record{
			Key:   fmt.Sprintf("a%d", i),
			Value: []byte{},
		})
	}
	if results, err := s.Read("a", store.ReadLimit(5), store.ReadPrefix()); err != nil {
		t.Fatal(err)
	} else {
		if len(results) != 5 {
			t.Fatal("Expected 5 results, got ", len(results))
		}
		if results[0].Key != "a0" {
			t.Fatalf("Expected a0, got %s", results[0].Key)
		}
		if results[4].Key != "a4" {
			t.Fatalf("Expected a4, got %s", results[4].Key)
		}
	}
	if results, err := s.Read("a", store.ReadLimit(30), store.ReadOffset(5), store.ReadPrefix()); err != nil {
		t.Fatal(err)
	} else {
		if len(results) != 5 {
			t.Fatal("Expected 5 results, got ", len(results))
		}
	}
}

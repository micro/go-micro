package natsjskv

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go-micro.dev/v5/store"
)

func TestNats(t *testing.T) {
	// Setup without calling Init on purpose
	var err error
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		addr := startNatsServer(ctx, t)
		s := NewStore(store.Nodes(addr), EncodeKeys())

		// Test String method
		t.Log("Testing:", s.String())

		err = basicTest(t, s)
		if err != nil {
			t.Log(err)
			continue
		}

		// Test reading non-existing key
		r, err := s.Read("this-is-a-random-key")
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Expected %# v, got %# v", store.ErrNotFound, err)
		}
		if len(r) > 0 {
			t.Fatal("Lenth should be 0")
		}
		err = s.Close()
		if err != nil {
			t.Logf("Failed to close store: %v", err)
		}
		cancel()
		return
	}
	t.Fatal(err)
}

func TestOptions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := testSetup(ctx, t,
		DefaultMemory(),

		// Having a non-default description will trigger nats.ErrStreamNameAlreadyInUse
		//  since the buckets have been created in previous tests with a different description.
		//
		// NOTE: this is only the case with a manually set up server, not with current
		//       test setup, where new servers are started for each test.
		DefaultDescription("My fancy description"),

		// Option has no effect in this context, just to test setting the option
		JetStreamOptions(nats.PublishAsyncMaxPending(256)),

		// Sets a custom NATS client name, just to test the NatsOptions() func
		NatsOptions(nats.Options{Name: "Go NATS Store Plugin Tests Client"}),

		KeyValueOptions(&nats.KeyValueConfig{
			Bucket:      "TestBucketName",
			Description: "This bucket is not used",
			TTL:         5 * time.Minute,
			MaxBytes:    1024,
			Storage:     nats.MemoryStorage,
			Replicas:    1,
		}),

		// Encode keys to avoid character limitations
		EncodeKeys(),
	)
	defer cancel()

	if err := basicTest(t, s); err != nil {
		t.Fatal(err)
	}
}

func TestTTL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ttl := 500 * time.Millisecond
	s := testSetup(ctx, t,
		DefaultTTL(ttl),

		// Since these buckets will be new they will have the new description
		DefaultDescription("My fancy description"),
	)
	defer cancel()

	// Use a uuid to make sure a new bucket is created when using local server
	id := uuid.New().String()
	for _, r := range table {
		if err := s.Write(r.Record, store.WriteTo(r.Database+id, r.Table)); err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(ttl * 2)

	for _, r := range table {
		res, err := s.Read(r.Record.Key, store.ReadFrom(r.Database+id, r.Table))
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Expected %# v, got %# v", store.ErrNotFound, err)
		}
		if len(res) > 0 {
			t.Fatal("Fetched record while it should have expired")
		}
	}
}

func TestMetaData(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := testSetup(ctx, t)
	defer cancel()

	record := store.Record{
		Key:   "KeyOne",
		Value: []byte("Some value"),
		Metadata: map[string]interface{}{
			"meta-one": "val",
			"meta-two": 5,
		},
		Expiry: 0,
	}
	bucket := "meta-data-test"
	if err := s.Write(&record, store.WriteTo(bucket, "")); err != nil {
		t.Fatal(err)
	}

	r, err := s.Read(record.Key, store.ReadFrom(bucket, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(r) == 0 {
		t.Fatal("No results found")
	}

	m := r[0].Metadata
	if m["meta-one"].(string) != record.Metadata["meta-one"].(string) ||
		m["meta-two"].(float64) != float64(record.Metadata["meta-two"].(int)) {
		t.Fatalf("Metadata does not match: (%+v) != (%+v)", m, record.Metadata)
	}
}

func TestDelete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := testSetup(ctx, t)
	defer cancel()

	for _, r := range table {
		if err := s.Write(r.Record, store.WriteTo(r.Database, r.Table)); err != nil {
			t.Fatal(err)
		}

		if err := s.Delete(r.Record.Key, store.DeleteFrom(r.Database, r.Table)); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)

		res, err := s.Read(r.Record.Key, store.ReadFrom(r.Database, r.Table))
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Expected %# v, got %# v", store.ErrNotFound, err)
		}
		if len(res) > 0 {
			t.Fatalf("Failed to delete %s:%s from %s %s (len: %d)", r.Record.Key, r.Record.Value, r.Database, r.Table, len(res))
		}
	}
}

func TestList(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := testSetup(ctx, t)
	defer cancel()

	for _, r := range table {
		if err := s.Write(r.Record, store.WriteTo(r.Database, r.Table)); err != nil {
			t.Fatal(err)
		}
	}

	l := []struct {
		Database string
		Table    string
		Length   int
		Prefix   string
		Suffix   string
		Offset   int
		Limit    int
	}{
		{Length: 7},
		{Database: "prefix-test", Length: 7},
		{Database: "prefix-test", Offset: 2, Length: 5},
		{Database: "prefix-test", Offset: 2, Limit: 3, Length: 3},
		{Database: "prefix-test", Table: "names", Length: 3},
		{Database: "prefix-test", Table: "cities", Length: 4},
		{Database: "prefix-test", Table: "cities", Suffix: "City", Length: 3},
		{Database: "prefix-test", Table: "cities", Suffix: "City", Limit: 2, Length: 2},
		{Database: "prefix-test", Table: "cities", Suffix: "City", Offset: 1, Length: 2},
		{Prefix: "test", Length: 1},
		{Table: "some_table", Prefix: "test", Suffix: "test", Length: 2},
	}

	for i, entry := range l {
		// Test listing keys
		keys, err := s.List(
			store.ListFrom(entry.Database, entry.Table),
			store.ListPrefix(entry.Prefix),
			store.ListSuffix(entry.Suffix),
			store.ListOffset(uint(entry.Offset)),
			store.ListLimit(uint(entry.Limit)),
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != entry.Length {
			t.Fatalf("Length of returned keys is invalid for test %d - %+v (%d)", i+1, entry, len(keys))
		}

		// Test reading keys
		if entry.Prefix != "" || entry.Suffix != "" {
			var key string
			options := []store.ReadOption{
				store.ReadFrom(entry.Database, entry.Table),
				store.ReadLimit(uint(entry.Limit)),
				store.ReadOffset(uint(entry.Offset)),
			}
			if entry.Prefix != "" {
				key = entry.Prefix
				options = append(options, store.ReadPrefix())
			}
			if entry.Suffix != "" {
				key = entry.Suffix
				options = append(options, store.ReadSuffix())
			}
			r, err := s.Read(key, options...)
			if err != nil {
				t.Fatal(err)
			}
			if len(r) != entry.Length {
				t.Fatalf("Length of read keys is invalid for test %d - %+v (%d)", i+1, entry, len(r))
			}
		}
	}
}

func TestDeleteBucket(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := testSetup(ctx, t)
	defer cancel()

	for _, r := range table {
		if err := s.Write(r.Record, store.WriteTo(r.Database, r.Table)); err != nil {
			t.Fatal(err)
		}
	}

	bucket := "prefix-test"
	if err := s.Delete(bucket, DeleteBucket()); err != nil {
		t.Fatal(err)
	}

	keys, err := s.List(store.ListFrom(bucket, ""))
	if err != nil && !errors.Is(err, ErrBucketNotFound) {
		t.Fatalf("Failed to delete bucket: %v", err)
	}

	if len(keys) > 0 {
		t.Fatal("Length of key list should be 0 after bucket deletion")
	}

	r, err := s.Read("", store.ReadPrefix(), store.ReadFrom(bucket, ""))
	if err != nil && !errors.Is(err, ErrBucketNotFound) {
		t.Fatalf("Failed to delete bucket: %v", err)
	}
	if len(r) > 0 {
		t.Fatal("Length of record list should be 0 after bucket deletion", len(r))
	}
}

func TestEnforceLimits(t *testing.T) {
	s := []string{"a", "b", "c", "d"}
	var testCasts = []struct {
		Alias    string
		Offset   uint
		Limit    uint
		Expected []string
	}{
		{"plain", 0, 0, []string{"a", "b", "c", "d"}},
		{"offset&limit-1", 1, 3, []string{"b", "c", "d"}},
		{"offset&limit-2", 1, 1, []string{"b"}},
		{"offset=length", 4, 0, []string{}},
		{"offset>length", 222, 0, []string{}},
		{"limit>length", 0, 36, []string{"a", "b", "c", "d"}},
	}
	for _, tc := range testCasts {
		actual := enforceLimits(s, tc.Limit, tc.Offset)
		if !reflect.DeepEqual(actual, tc.Expected) {
			t.Fatalf("%s: Expected %v, got %v", tc.Alias, tc.Expected, actual)
		}
	}
}

func basicTest(t *testing.T, s store.Store) error {
	t.Helper()
	for _, test := range table {
		if err := s.Write(test.Record, store.WriteTo(test.Database, test.Table)); err != nil {
			return errors.Wrap(err, "Failed to write record in basic test")
		}
		r, err := s.Read(test.Record.Key, store.ReadFrom(test.Database, test.Table))
		if err != nil {
			return errors.Wrap(err, "Failed to read record in basic test")
		}
		if len(r) == 0 {
			t.Fatalf("No results found for %s (%s) %s", test.Record.Key, test.Database, test.Table)
		}

		key := test.Record.Key
		val1 := string(test.Record.Value)

		key2 := r[0].Key
		val2 := string(r[0].Value)
		if val1 != val2 {
			t.Fatalf("Value not equal for (%s: %s) != (%s: %s)", key, val1, key2, val2)
		}
	}
	return nil
}

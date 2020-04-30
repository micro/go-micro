package cockroach

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kr/pretty"
	"github.com/micro/go-micro/v2/store"
)

func TestSQL(t *testing.T) {
	if len(os.Getenv("IN_TRAVIS_CI")) != 0 {
		t.Skip()
	}

	connection := fmt.Sprintf(
		"host=%s port=%d user=%s sslmode=disable dbname=%s",
		"localhost",
		26257,
		"root",
		"test",
	)
	db, err := sql.Open("postgres", connection)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		t.Skip("store/cockroach: can't connect to db")
	}
	db.Close()

	sqlStore := NewStore(
		store.Database("testsql"),
		store.Nodes(connection),
	)

	if err := sqlStore.Init(); err != nil {
		t.Fatal(err)
	}

	keys, err := sqlStore.List()
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("%# v\n", pretty.Formatter(keys))
	}

	err = sqlStore.Write(
		&store.Record{
			Key:   "test",
			Value: []byte("foo"),
		},
	)
	if err != nil {
		t.Error(err)
	}
	err = sqlStore.Write(
		&store.Record{
			Key:   "bar",
			Value: []byte("baz"),
		},
	)
	if err != nil {
		t.Error(err)
	}
	err = sqlStore.Write(
		&store.Record{
			Key:   "qux",
			Value: []byte("aasad"),
		},
	)
	if err != nil {
		t.Error(err)
	}
	err = sqlStore.Delete("qux")
	if err != nil {
		t.Error(err)
	}

	err = sqlStore.Write(&store.Record{
		Key:    "test",
		Value:  []byte("bar"),
		Expiry: time.Second * 10,
	})
	if err != nil {
		t.Error(err)
	}

	records, err := sqlStore.Read("test")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%# v\n", pretty.Formatter(records))
	if string(records[0].Value) != "bar" {
		t.Error("Expected bar, got ", string(records[0].Value))
	}

	time.Sleep(11 * time.Second)
	_, err = sqlStore.Read("test")
	switch err {
	case nil:
		t.Error("Key test should have expired")
	default:
		t.Error(err)
	case store.ErrNotFound:
		break
	}
	sqlStore.Delete("bar")
	sqlStore.Write(&store.Record{Key: "aaa", Value: []byte("bbb"), Expiry: 5 * time.Second})
	sqlStore.Write(&store.Record{Key: "aaaa", Value: []byte("bbb"), Expiry: 5 * time.Second})
	sqlStore.Write(&store.Record{Key: "aaaaa", Value: []byte("bbb"), Expiry: 5 * time.Second})
	results, err := sqlStore.Read("a", store.ReadPrefix())
	if err != nil {
		t.Error(err)
	}
	if len(results) != 3 {
		t.Fatal("Results should have returned 3 records")
	}
	time.Sleep(6 * time.Second)
	results, err = sqlStore.Read("a", store.ReadPrefix())
	if err != nil {
		t.Error(err)
	}
	if len(results) != 0 {
		t.Fatal("Results should have returned 0 records")
	}
}

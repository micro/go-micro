package postgresql

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/kr/pretty"
	"github.com/micro/go-micro/store"
)

func TestSQL(t *testing.T) {
	connection := fmt.Sprintf(
		"host=%s port=%d user=%s sslmode=disable dbname=%s",
		"localhost",
		5432,
		"jake",
		"test",
	)
	db, err := sql.Open("postgres", connection)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		t.Skip(err)
	}
	db.Close()

	sqlStore, err := New(
		store.Namespace("testsql"),
		store.Nodes(connection),
	)
	if err != nil {
		t.Fatal(err.Error())
	}

	records, err := sqlStore.List()
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("%# v\n", pretty.Formatter(records))
	}

	err = sqlStore.Write(
		&store.Record{
			Key:   "test",
			Value: []byte("foo"),
		},
		&store.Record{
			Key:   "bar",
			Value: []byte("baz"),
		},
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
		Expiry: time.Minute,
	})
	if err != nil {
		t.Error(err)
	}

	records, err = sqlStore.Read("test")
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("%# v\n", pretty.Formatter(records))
		if string(records[0].Value) != "bar" {
			t.Error("Expected bar, got ", string(records[0].Value))
		}
	}

	time.Sleep(61 * time.Second)
	records, err = sqlStore.Read("test")
	if err == nil {
		t.Error("Key test should have expired")
	} else {
		if err != store.ErrNotFound {
			t.Error(err)
		}
	}
}

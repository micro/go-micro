package mysql

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/asim/go-micro/v3/store"
)

var (
	sqlStoreT store.Store
)

func TestMain(m *testing.M) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		os.Exit(0)
	}

	sqlStoreT = NewStore(
		store.Database("testMicro"),
		store.Nodes("root:123@(127.0.0.1:3306)/test?charset=utf8&parseTime=true&loc=Asia%2FShanghai"),
	)
	os.Exit(m.Run())
}

func TestWrite(t *testing.T) {
	err := sqlStoreT.Write(
		&store.Record{
			Key:    "test",
			Value:  []byte("foo2"),
			Expiry: time.Second * 200,
		},
	)
	if err != nil {
		t.Error(err)
	}
}

func TestDelete(t *testing.T) {
	err := sqlStoreT.Delete("test")
	if err != nil {
		t.Error(err)
	}
}

func TestRead(t *testing.T) {
	records, err := sqlStoreT.Read("test")
	if err != nil {
		t.Error(err)
	}

	t.Log(string(records[0].Value))
}

func TestList(t *testing.T) {
	records, err := sqlStoreT.List()
	if err != nil {
		t.Error(err)
	} else {
		beauty, _ := json.Marshal(records)
		t.Log(string(beauty))
	}
}

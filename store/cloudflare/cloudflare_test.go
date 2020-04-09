package cloudflare

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/store"
)

func TestCloudflare(t *testing.T) {
	if len(os.Getenv("IN_TRAVIS_CI")) != 0 {
		t.Skip()
	}

	apiToken, accountID := os.Getenv("CF_API_TOKEN"), os.Getenv("CF_ACCOUNT_ID")
	kvID := os.Getenv("KV_NAMESPACE_ID")
	if len(apiToken) == 0 || len(accountID) == 0 || len(kvID) == 0 {
		t.Skip("No Cloudflare API keys available, skipping test")
	}
	rand.Seed(time.Now().UnixNano())
	randomK := strconv.Itoa(rand.Int())
	randomV := strconv.Itoa(rand.Int())

	wkv := NewStore(
		Token(apiToken),
		Account(accountID),
		Namespace(kvID),
		CacheTTL(60000000000),
	)

	records, err := wkv.List()
	if err != nil {
		t.Fatalf("List: %s\n", err.Error())
	} else {
		if len(os.Getenv("IN_TRAVIS_CI")) == 0 {
			t.Log("Listed " + strconv.Itoa(len(records)) + " records")
		}
	}

	err = wkv.Write(&store.Record{
		Key:   randomK,
		Value: []byte(randomV),
	})
	if err != nil {
		t.Errorf("Write: %s", err.Error())
	}
	err = wkv.Write(&store.Record{
		Key:    "expirationtest",
		Value:  []byte("This message will self destruct"),
		Expiry: 75 * time.Second,
	})
	if err != nil {
		t.Errorf("Write: %s", err.Error())
	}

	// This might be needed for cloudflare eventual consistency
	time.Sleep(1 * time.Minute)

	r, err := wkv.Read(randomK)
	if err != nil {
		t.Errorf("Read: %s\n", err.Error())
	}
	if len(r) != 1 {
		t.Errorf("Expected to read 1 key, got %d keys\n", len(r))
	}
	if string(r[0].Value) != randomV {
		t.Errorf("Read: expected %s, got %s\n", randomK, string(r[0].Value))
	}

	r, err = wkv.Read("expirationtest")
	if err != nil {
		t.Errorf("Read: expirationtest should still exist")
	}
	if r[0].Expiry == 0 {
		t.Error("Expected r to have an expiry")
	} else {
		t.Log(r[0].Expiry)
	}

	time.Sleep(20 * time.Second)
	r, err = wkv.Read("expirationtest")
	if err == nil && len(r) != 0 {
		t.Error("Read: Managed to read expirationtest, but it should have expired")
		t.Log(err, r[0].Key, string(r[0].Value), r[0].Expiry, len(r))
	}

	err = wkv.Delete(randomK)
	if err != nil {
		t.Errorf("Delete: %s\n", err.Error())
	}

}

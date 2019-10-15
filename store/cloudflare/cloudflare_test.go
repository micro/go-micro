package cloudflare

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/store"
)

func TestCloudflare(t *testing.T) {
	apiToken, accountID := os.Getenv("CF_API_TOKEN"), os.Getenv("CF_ACCOUNT_ID")
	kvID := os.Getenv("KV_NAMESPACE_ID")
	if len(apiToken) == 0 || len(accountID) == 0 || len(kvID) == 0 {
		t.Skip("No Cloudflare API keys available, skipping test")
	}
	rand.Seed(time.Now().UnixNano())
	randomK := strconv.Itoa(rand.Int())
	randomV := strconv.Itoa(rand.Int())

	wkv, err := New(
		options.WithValue("CF_API_TOKEN", apiToken),
		options.WithValue("CF_ACCOUNT_ID", accountID),
		options.WithValue("KV_NAMESPACE_ID", kvID),
	)

	if err != nil {
		t.Fatal(err.Error())
	}

	_, err = wkv.Sync()
	if err != nil {
		t.Fatalf("Sync: %s\n", err.Error())
	}

	err = wkv.Write(&store.Record{
		Key:   randomK,
		Value: []byte(randomV),
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

	err = wkv.Delete(randomK)
	if err != nil {
		t.Errorf("Delete: %s\n", err.Error())
	}

}

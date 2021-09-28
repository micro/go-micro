package consul

import (
	"os"
	"strings"
	gosync "sync"
	"testing"

	"github.com/asim/go-micro/v3/sync"
)

const (
	defaultHost = "localhost:8500"
	defaultLock = "micro-sync-lock"
)

func TestSyncLock(t *testing.T) {
	var lockWg gosync.WaitGroup
	var testWg gosync.WaitGroup
	testWg.Add(1)
	lockWg.Add(1)

	// get settings from env
	consulHost := os.Getenv("MICRO_TEST_CONSUL")
	if consulHost == "" {
		consulHost = defaultHost
	}

	lockID := os.Getenv("MICRO_TEST_LOCK_ID")
	if lockID == "" {
		lockID = defaultLock
	}

	// create a new sync manager
	s := NewSync(sync.Nodes(consulHost))

	// wait for the lock to be acquired before unlocking
	go func() {
		t.Log("Waiting for lock", lockID)
		lockWg.Wait()
		t.Log("Unlocking", lockID)
		if err := s.Unlock(lockID); err != nil {
			t.Error(err)
		}
		t.Log("Unlocked", lockID)
		testWg.Done()
	}()

	// generate a lock
	t.Log("Locking", lockID)
	if err := s.Lock(lockID); err != nil {
		if strings.HasSuffix(err.Error(), "connection refused") {
			t.Log("Failed to connect to consul, skipping test. Please start a consul instance to perform this test")
			lockWg.Done()
			return
		}
		t.Error(err)
		lockWg.Done()
		return
	}

	t.Log("Locked", lockID)

	lockWg.Done()
	testWg.Wait()

	t.Log("Test complete")
}

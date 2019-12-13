// Package memory provides a sync.Mutex implementation of the lock for local use
package memory

import (
	"sync"
	"time"

	lock "github.com/micro/go-micro/sync/lock"
)

type memoryLock struct {
	sync.RWMutex
	locks map[string]*mlock
}

type mlock struct {
	id      string
	time    time.Time
	ttl     time.Duration
	release chan bool
}

func (m *memoryLock) Acquire(id string, opts ...lock.AcquireOption) error {
	// lock our access
	m.Lock()

	var options lock.AcquireOptions
	for _, o := range opts {
		o(&options)
	}

	lk, ok := m.locks[id]
	if !ok {
		m.locks[id] = &mlock{
			id:      id,
			time:    time.Now(),
			ttl:     options.TTL,
			release: make(chan bool),
		}
		// unlock
		m.Unlock()
		return nil
	}

	m.Unlock()

	// set wait time
	var wait <-chan time.Time
	var ttl <-chan time.Time

	// decide if we should wait
	if options.Wait > time.Duration(0) {
		wait = time.After(options.Wait)
	}

	// check the ttl of the lock
	if lk.ttl > time.Duration(0) {
		// time lived for the lock
		live := time.Since(lk.time)

		// set a timer for the leftover ttl
		if live > lk.ttl {
			// release the lock if it expired
			_ = m.Release(id)
		} else {
			ttl = time.After(live)
		}
	}

lockLoop:
	for {
		// wait for the lock to be released
		select {
		case <-lk.release:
			m.Lock()

			// someone locked before us
			lk, ok = m.locks[id]
			if ok {
				m.Unlock()
				continue
			}

			// got chance to lock
			m.locks[id] = &mlock{
				id:      id,
				time:    time.Now(),
				ttl:     options.TTL,
				release: make(chan bool),
			}

			m.Unlock()

			break lockLoop
		case <-ttl:
			// ttl exceeded
			_ = m.Release(id)
			// TODO: check the ttl again above
			ttl = nil
			// try acquire
			continue
		case <-wait:
			return lock.ErrLockTimeout
		}
	}

	return nil
}

func (m *memoryLock) Release(id string) error {
	m.Lock()
	defer m.Unlock()

	lk, ok := m.locks[id]
	// no lock exists
	if !ok {
		return nil
	}

	// delete the lock
	delete(m.locks, id)

	select {
	case <-lk.release:
		return nil
	default:
		close(lk.release)
	}

	return nil
}

func NewLock(opts ...lock.Option) lock.Lock {
	var options lock.Options
	for _, o := range opts {
		o(&options)
	}

	return &memoryLock{
		locks: make(map[string]*mlock),
	}
}

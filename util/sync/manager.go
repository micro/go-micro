package sync

import (
	"time"

	"github.com/micro/go-micro/v2/store"
	"github.com/pkg/errors"
)

type operation struct {
	operation action
	record    *store.Record
	deadline  time.Time
	retries   int
	maxiumum  int
}

// action represents the type of a queued operation
type action int

const (
	readOp action = iota + 1
	writeOp
	deleteOp
	listOp
)

func (c *syncStore) syncManager() {
	tickerAggregator := make(chan struct{ index int })
	for i, ticker := range c.pendingWriteTickers {
		go func(index int, c chan struct{ index int }, t *time.Ticker) {
			for range t.C {
				c <- struct{ index int }{index: index}
			}
		}(i, tickerAggregator, ticker)
	}
	for {
		select {
		case i := <-tickerAggregator:
			println(i.index, "ticked")
			c.processQueue(i.index)
		}
	}
}

func (c *syncStore) processQueue(index int) {
	c.Lock()
	defer c.Unlock()
	q := c.pendingWrites[index]
	for i := 0; i < q.Len(); i++ {
		r, ok := q.PopFront()
		if !ok {
			panic(errors.Errorf("retrieved an invalid value from the L%d sync queue", index+1))
		}
		ir, ok := r.(*internalRecord)
		if !ok {
			panic(errors.Errorf("retrieved a non-internal record from the L%d sync queue", index+1))
		}
		if !ir.expiresAt.IsZero() && time.Now().After(ir.expiresAt) {
			continue
		}
		nr := &store.Record{
			Key: ir.key,
		}
		nr.Value = make([]byte, len(ir.value))
		copy(nr.Value, ir.value)
		if !ir.expiresAt.IsZero() {
			nr.Expiry = time.Until(ir.expiresAt)
		}
		// Todo = internal queue also has to hold the corresponding store.WriteOptions
		if err := c.syncOpts.Stores[index+1].Write(nr); err != nil {
			// some error, so queue for retry and bail
			q.PushBack(ir)
			return
		}
	}
}

func intpow(x, y int64) int64 {
	result := int64(1)
	for 0 != y {
		if 0 != (y & 1) {
			result *= x
		}
		y >>= 1
		x *= x
	}
	return result
}

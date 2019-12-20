package kubernetes

import "github.com/micro/go-micro/debug/log"

// ByTimestamp lets you sort log records by Timestamp (implements Sort.Sort)
type byTimestamp []log.Record

// Len returns the number of Log records (implements Sort.Sort)
func (b byTimestamp) Len() int { return len(b) }

// Swap swaps 2 Log records (implements Sort.Sort)
func (b byTimestamp) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Less checks if a record was before another record (implements Sort.Sort)
func (b byTimestamp) Less(i, j int) bool { return b[i].Timestamp.Before(b[j].Timestamp) }

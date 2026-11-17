package nano

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// ErrAlreadyClaimed is returned when a block hash or an order id has already
// paid for a request. One block pays for one order, once.
var ErrAlreadyClaimed = errors.New("nano: payment already claimed")

// ErrClaimsLocked is returned by OpenFileClaimStore when another store, in
// this process or another one, already holds the claims file.
var ErrClaimsLocked = errors.New("nano: claims file is in use by another instance")

var blockHashPattern = regexp.MustCompile(`^[0-9A-F]{64}$`)

// ClaimStore records which block paid which order. Claim must be atomic: it
// either records both keys or returns an error and records nothing.
type ClaimStore interface {
	Claim(blockHash, orderID string) error
}

// FileClaimStore keeps claims in an append-only file, one
// "<BLOCK_HASH> <order_id>" line each, written and fsynced before Claim
// returns. The file is read back on open, so a restart does not make an old
// block spendable again.
//
// The store holds an exclusive lock on the file for its lifetime (flock on
// Unix, LockFileEx on Windows), so a second store on the same file, in this
// process or another gateway, fails to open with ErrClaimsLocked instead of
// accepting the same block twice. Run one gateway per claims file.
type FileClaimStore struct {
	mu     sync.Mutex
	f      *os.File
	size   int64
	err    error // sticky: set once the file may no longer match memory
	sync   func() error
	blocks map[string]struct{}
	orders map[string]struct{}
}

// OpenFileClaimStore opens (or creates, mode 0600) the claims file at path and
// loads every claim already in it. A malformed complete line is an error, not
// a skip. A trailing fragment without a newline is what an interrupted append
// leaves behind: Claim never reported it as recorded, so it is cut off and the
// next append starts on a fresh line.
func OpenFileClaimStore(path string) (*FileClaimStore, error) {
	f, created, err := openClaimsFile(path)
	if err != nil {
		return nil, err
	}
	if err := lockFile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("%w: %s: %v", ErrClaimsLocked, path, err)
	}
	if created {
		// Make the new directory entry durable, not only the file's contents.
		if err := syncDir(filepath.Dir(path)); err != nil {
			f.Close()
			return nil, err
		}
	}
	s := &FileClaimStore{f: f, blocks: map[string]struct{}{}, orders: map[string]struct{}{}}
	s.sync = f.Sync
	if err := s.load(path); err != nil {
		f.Close()
		return nil, err
	}
	return s, nil
}

func openClaimsFile(path string) (*os.File, bool, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL|os.O_APPEND, 0o600)
	if err == nil {
		return f, true, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return nil, false, err
	}
	f, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0o600)
	return f, false, err
}

func (s *FileClaimStore) load(path string) error {
	data, err := io.ReadAll(s.f)
	if err != nil {
		return err
	}
	complete := data
	if i := bytes.LastIndexByte(data, '\n'); i+1 < len(data) {
		complete = data[:i+1]
	}
	line := 0
	for _, raw := range strings.SplitAfter(string(complete), "\n") {
		if raw == "" {
			continue
		}
		line++
		text := strings.TrimSpace(raw)
		if text == "" {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) != 2 || !blockHashPattern.MatchString(fields[0]) || !ValidOrderID(fields[1]) {
			return fmt.Errorf("nano: claims file %s line %d is malformed", path, line)
		}
		s.blocks[fields[0]] = struct{}{}
		s.orders[fields[1]] = struct{}{}
	}
	s.size = int64(len(complete))
	if len(complete) < len(data) {
		if err := s.f.Truncate(s.size); err != nil {
			return fmt.Errorf("nano: claims file %s: dropping incomplete last record: %w", path, err)
		}
		if err := s.f.Sync(); err != nil {
			return err
		}
	}
	return nil
}

// Claim records blockHash -> orderID, refusing a block or an order that was
// claimed before.
//
// If the record was written but the fsync failed, the claim may or may not be
// on disk. Claim then keeps it as taken in memory, so the block and order
// cannot be claimed again, returns the error, and refuses every later claim
// (fail closed) until the store is reopened and the file read back.
func (s *FileClaimStore) Claim(blockHash, orderID string) error {
	blockHash = strings.ToUpper(blockHash)
	if !blockHashPattern.MatchString(blockHash) || !ValidOrderID(orderID) {
		return errors.New("nano: malformed claim")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.blocks[blockHash]; ok {
		return ErrAlreadyClaimed
	}
	if _, ok := s.orders[orderID]; ok {
		return ErrAlreadyClaimed
	}
	if s.err != nil {
		return s.err
	}
	n, err := s.f.WriteString(blockHash + " " + orderID + "\n")
	if err != nil {
		// Cut any partial record so the file stays line-aligned. If even that
		// fails, the file no longer matches memory: stop accepting claims.
		if terr := s.f.Truncate(s.size); terr != nil {
			s.err = fmt.Errorf("nano: claims file unusable after failed write: %w", err)
		}
		return err
	}
	s.size += int64(n)
	s.blocks[blockHash] = struct{}{}
	s.orders[orderID] = struct{}{}
	if err := s.sync(); err != nil {
		s.err = fmt.Errorf("nano: claims file fsync failed: %w", err)
		return s.err
	}
	return nil
}

// Close releases the lock and closes the claims file.
func (s *FileClaimStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.f.Close()
}

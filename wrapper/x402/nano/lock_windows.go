//go:build windows

package nano

import (
	"os"
	"syscall"
	"unsafe"
)

var procLockFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("LockFileEx")

const (
	lockfileFailImmediately = 0x1
	lockfileExclusiveLock   = 0x2
)

// lockFile takes an exclusive, non-blocking LockFileEx lock on f. The locked
// byte is far past any real data, so it never blocks reads or appends; it only
// keeps a second store from locking the same file. It is released when f is
// closed.
func lockFile(f *os.File) error {
	ol := &syscall.Overlapped{Offset: 0xFFFFFFFF, OffsetHigh: 0x7FFFFFFF}
	r, _, err := procLockFileEx.Call(
		f.Fd(),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r == 0 {
		return err
	}
	return nil
}

// syncDir is a no-op on Windows: a directory handle from os.Open cannot be
// flushed there, and NTFS journals the new directory entry itself.
func syncDir(string) error { return nil }

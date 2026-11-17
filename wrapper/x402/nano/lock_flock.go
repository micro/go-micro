//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package nano

import (
	"os"
	"syscall"
)

// lockFile takes an exclusive, non-blocking flock on f. It is released when f
// is closed.
func lockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

// syncDir fsyncs a directory so a newly created entry in it is durable.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

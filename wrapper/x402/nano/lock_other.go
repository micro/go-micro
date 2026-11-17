//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly || windows)

package nano

import "os"

// lockFile is a no-op on platforms without flock or LockFileEx; there the
// claims file must be used by a single gateway instance only.
func lockFile(*os.File) error { return nil }

// syncDir fsyncs a directory so a newly created entry in it is durable.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

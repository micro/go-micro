package file_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-micro.dev/v5/config/source"
	"go-micro.dev/v5/config/source/file"
)

// createTestFile a local helper to creates a temporary file with the given data
func createTestFile(data []byte) (*os.File, func(), string, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("file.%d", time.Now().UnixNano()))
	fh, err := os.Create(path)
	if err != nil {
		return nil, func() {}, "", err
	}

	_, err = fh.Write(data)
	if err != nil {
		return nil, func() {}, "", err
	}

	return fh, func() {
		fh.Close()
		os.Remove(path)
	}, path, err
}

func TestWatcher(t *testing.T) {
	data := []byte(`{"foo": "bar"}`)
	fh, cleanup, path, err := createTestFile(data)
	if err != nil {
		t.Error(err)
	}
	defer cleanup()

	f := file.NewSource(file.WithPath(path))
	if err != nil {
		t.Error(err)
	}

	// create a watcher
	w, err := f.Watch()
	if err != nil {
		t.Error(err)
	}

	newdata := []byte(`{"foo": "baz"}`)

	go func() {
		sc, err := w.Next()
		if err != nil {
			t.Error(err)
			return
		}

		if !bytes.Equal(sc.Data, newdata) {
			t.Error("expected data to be different")
		}
	}()

	// rewrite to the file to trigger a change
	_, err = fh.WriteAt(newdata, 0)
	if err != nil {
		t.Error(err)
	}

	// wait for the underlying watcher to detect changes
	time.Sleep(time.Second)
}

func TestWatcherStop(t *testing.T) {
	data := []byte(`{"foo": "bar"}`)
	_, cleanup, path, err := createTestFile(data)
	if err != nil {
		t.Error(err)
	}
	defer cleanup()

	src := file.NewSource(file.WithPath(path))
	if err != nil {
		t.Error(err)
	}

	// create a watcher
	w, err := src.Watch()
	if err != nil {
		t.Error(err)
	}

	defer func() {
		var err error
		c := make(chan struct{})
		defer close(c)

		go func() {
			_, err = w.Next()
			c <- struct{}{}
		}()

		select {
		case <-time.After(2 * time.Second):
			err = errors.New("timeout waiting for Watcher.Next() to return")
		case <-c:
		}

		if !errors.Is(err, source.ErrWatcherStopped) {
			t.Error(err)
		}
	}()

	// stop the watcher
	w.Stop()
}

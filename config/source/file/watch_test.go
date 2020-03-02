package file_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/config"
	"github.com/micro/go-micro/v2/config/source"
	"github.com/micro/go-micro/v2/config/source/file"
)

func TestWatch(t *testing.T) {
	data := []byte(`{"foo": "bar"}`)
	path := filepath.Join(os.TempDir(), fmt.Sprintf("file_%d.json", time.Now().UnixNano()))
	fh, err := os.Create(path)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		fh.Close()
		os.Remove(path)
	}()

	_, err = fh.Write(data)
	if err != nil {
		t.Error(err)
	}

	config.Load(file.NewSource(
		file.WithPath(path),
	))

	w, err := config.Watch()
	if err != nil {
		t.Error(err)
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		w.Stop()
	}()

	t.Log(config.Get("foo").String("no_foo"))

	time.Sleep(50 * time.Millisecond)
	_, err = fh.WriteAt([]byte(`{"foo": "bar2"}`), 0)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(50 * time.Millisecond)
	t.Log(config.Get("foo").String("no_foo"))
}

func TestDisableUpdates(t *testing.T) {
	data := []byte(`{"foo": "bar"}`)
	path := filepath.Join(os.TempDir(), fmt.Sprintf("file_%d.json", time.Now().UnixNano()))
	fh, err := os.Create(path)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		fh.Close()
		os.Remove(path)
	}()

	_, err = fh.Write(data)
	if err != nil {
		t.Error(err)
	}

	config.Load(file.NewSource(
		file.WithPath(path),
		source.DisableUpdates(),
	))

	w, err := config.Watch()
	if err != nil {
		t.Error(err)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		w.Stop()
	}()

	t.Log(config.Get("foo").String("no_foo"))

	time.Sleep(50 * time.Millisecond)
	_, err = fh.WriteAt([]byte(`{"foo": "bar2"}`), 0)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(50 * time.Millisecond)
	t.Log(config.Get("foo").String("no_foo"))
}

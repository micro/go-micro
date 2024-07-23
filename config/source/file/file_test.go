package file_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"go-micro.dev/v5/config"
	"go-micro.dev/v5/config/source/file"
)

func TestConfig(t *testing.T) {
	data := []byte(`{"foo": "bar"}`)
	path := filepath.Join(os.TempDir(), fmt.Sprintf("file.%d", time.Now().UnixNano()))
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

	conf, err := config.NewConfig()
	if err != nil {
		t.Fatal(err)
	}
	conf.Load(file.NewSource(file.WithPath(path)))
	// simulate multiple close
	go conf.Close()
	go conf.Close()
}

func TestFile(t *testing.T) {
	data := []byte(`{"foo": "bar"}`)
	path := filepath.Join(os.TempDir(), fmt.Sprintf("file.%d", time.Now().UnixNano()))
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

	f := file.NewSource(file.WithPath(path))
	c, err := f.Read()
	if err != nil {
		t.Error(err)
	}
	if string(c.Data) != string(data) {
		t.Logf("%+v", c)
		t.Error("data from file does not match")
	}
}

func TestWithFS(t *testing.T) {
	data := []byte(`{"foo": "bar"}`)
	path := fmt.Sprintf("file.%d", time.Now().UnixNano())

	fsMock := fstest.MapFS{
		path: &fstest.MapFile{
			Data: data,
			Mode: 0666,
		},
	}

	f := file.NewSource(file.WithFS(fsMock), file.WithPath(path))
	c, err := f.Read()
	if err != nil {
		t.Error(err)
	}
	if string(c.Data) != string(data) {
		t.Logf("%+v", c)
		t.Error("data from file does not match")
	}
}

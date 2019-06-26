package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/micro/go-micro/config/source/file"
)

func createFileForTest(t *testing.T) *os.File {
	data := []byte(`{"foo": "bar"}`)
	path := filepath.Join(os.TempDir(), fmt.Sprintf("file.%d", time.Now().UnixNano()))
	fh, err := os.Create(path)
	if err != nil {
		t.Error(err)
	}
	_, err = fh.Write(data)
	if err != nil {
		t.Error(err)
	}

	return fh
}

func TestLoadWithGoodFile(t *testing.T) {
	fh := createFileForTest(t)
	path := fh.Name()
	defer func() {
		fh.Close()
		os.Remove(path)
	}()

	// Create new config
	conf := NewConfig()
	// Load file source
	if err := conf.Load(file.NewSource(
		file.WithPath(path),
	)); err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}
}

func TestLoadWithInvalidFile(t *testing.T) {
	fh := createFileForTest(t)
	path := fh.Name()
	defer func() {
		fh.Close()
		os.Remove(path)
	}()

	// Create new config
	conf := NewConfig()
	// Load file source
	err := conf.Load(file.NewSource(
		file.WithPath(path),
		file.WithPath("/i/do/not/exists.json"),
	))

	if err == nil {
		t.Fatal("Expected error but none !")
	}
	if !strings.Contains(fmt.Sprintf("%v", err), "/i/do/not/exists.json") {
		t.Fatalf("Expected error to contain the unexisting file but got %v", err)
	}
}

func TestConsul(t *testing.T) {
	/*consulSource := consul.NewSource(
		// optionally specify consul address; default to localhost:8500
		consul.WithAddress("131.150.38.111:8500"),
		// optionally specify prefix; defaults to /micro/config
		consul.WithPrefix("/project"),
		// optionally strip the provided prefix from the keys, defaults to false
		consul.StripPrefix(true),
		consul.WithDatacenter("dc1"),
		consul.WithToken("xxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"),
	)

	// Create new config
	conf := NewConfig()

	// Load file source
	err := conf.Load(consulSource)
	if err != nil {
		t.Error(err)
		return
	}

	m := conf.Map()
	t.Log("m: ", m)

	v := conf.Get("project", "dc111", "port")

	t.Log("v: ", v.Int(13))*/

	t.Log("OK")
}

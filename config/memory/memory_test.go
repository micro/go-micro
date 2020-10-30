package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/asim/nitro/v3/config"
	"github.com/asim/nitro/v3/config/source/env"
	"github.com/asim/nitro/v3/config/source/file"
)

func createFileForIssue18(t *testing.T, content string) *os.File {
	data := []byte(content)
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

func TestConfigLoadWithGoodFile(t *testing.T) {
	fh := createFileForTest(t)
	path := fh.Name()
	defer func() {
		fh.Close()
		os.Remove(path)
	}()

	// Create new config
	conf, err := NewConfig(
		config.WithSource(file.NewSource(file.WithPath(path))),
	)
	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	// Load file source
	if err := conf.Sync(); err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}
}

func TestConfigLoadWithInvalidFile(t *testing.T) {
	fh := createFileForTest(t)
	path := fh.Name()
	defer func() {
		fh.Close()
		os.Remove(path)
	}()

	// Create new config
	_, err := NewConfig(config.WithSource(file.NewSource(
		file.WithPath(path),
		file.WithPath("/i/do/not/exists.json"),
	)))
	if err == nil {
		t.Fatal("Expected error but none !")
	}
	if !strings.Contains(fmt.Sprintf("%v", err), "/i/do/not/exists.json") {
		t.Fatalf("Expected error to contain the unexisting file but got %v", err)
	}
}

func TestConfigMerge(t *testing.T) {
	fh := createFileForIssue18(t, `{
  "amqp": {
    "host": "rabbit.platform",
    "port": 80
  },
  "handler": {
    "exchange": "springCloudBus"
  }
}`)
	path := fh.Name()
	defer func() {
		fh.Close()
		os.Remove(path)
	}()
	os.Setenv("AMQP_HOST", "rabbit.testing.com")

	conf, err := NewConfig(
		config.WithSource(file.NewSource(file.WithPath(path))),
		config.WithSource(env.NewSource()),
	)

	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}
	if err := conf.Sync(); err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	val, _ := conf.Load("amqp", "host")
	actualHost := val.String("backup")
	if actualHost != "rabbit.testing.com" {
		t.Fatalf("Expected %v but got %v",
			"rabbit.testing.com",
			actualHost)
	}
}

func equalS(t *testing.T, actual, expect string) {
	if actual != expect {
		t.Errorf("Expected %s but got %s", expect, actual)
	}
}

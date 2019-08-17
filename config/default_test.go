package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/config/source/env"
	"github.com/micro/go-micro/config/source/file"
)

var (
	sep = string(os.PathSeparator)
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
	conf := NewConfig()
	// Load file source
	if err := conf.Load(file.NewSource(
		file.WithPath(path),
	)); err != nil {
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

	conf := NewConfig()
	conf.Load(
		file.NewSource(
			file.WithPath(path),
		),
		env.NewSource(),
	)

	actualHost := conf.Get("amqp", "host").String("backup")
	if actualHost != "rabbit.testing.com" {
		t.Fatalf("Expected %v but got %v",
			"rabbit.testing.com",
			actualHost)
	}
}

func TestFileChange(t *testing.T) {
	// create a temp file
	fileName := uuid.New().String() + "testWatcher.json"
	f, err := os.OpenFile("."+sep+fileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
	defer os.Remove("." + sep + fileName)

	// load the file
	if err := Load(file.NewSource(
		file.WithPath("." + sep + fileName),
	)); err != nil {
		t.Error(err)
	}

	// watch changes
	watcher, err := Watch()
	if err != nil {
		t.Error(err)
	}
	changeTimes := 0
	go func() {
		for {
			v, err := watcher.Next()
			if err != nil {
				t.Error(err)
				return
			}
			changeTimes++
			t.Logf("file change，%s", string(v.Bytes()))
		}
	}()

	content := map[int]string{}
	// change the file
	for i := 0; i < 5; i++ {
		content[i] = time.Now().String()
		bytes, _ := json.Marshal(content)
		f.Truncate(0)
		f.Seek(0, 0)
		if _, err := f.Write(bytes); err != nil {
			t.Error(err)
		}

		time.Sleep(time.Second)
	}

	if changeTimes != 5 {
		t.Error(fmt.Errorf("watcher error: change times %d is not enough", changeTimes))
	}
}

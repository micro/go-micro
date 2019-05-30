package file

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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

	f := NewSource(WithPath(path))
	c, err := f.Read()
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v", c)
	if string(c.Data) != string(data) {
		t.Error("data from file does not match")
	}
}

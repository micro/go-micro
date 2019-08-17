package file

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/config"
)

var (
	sep = string(os.PathSeparator)
)

func TestChange(t *testing.T) {
	// create a temp file
	fileName := uuid.New().String() + "testWatcher.json"
	f, err := os.OpenFile("."+sep+fileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
	defer os.Remove("." + sep + fileName)

	// load the file
	if err := config.Load(NewSource(
		WithPath("." + sep + fileName),
	)); err != nil {
		t.Error(err)
	}

	// watch changes
	watcher, err := config.Watch()
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

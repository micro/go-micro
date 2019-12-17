package kubernetes

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/micro/go-micro/debug/log"
	"github.com/stretchr/testify/assert"
)

func TestKubernetes(t *testing.T) {
	k := New()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	s := os.Stderr
	os.Stderr = w
	meta := make(map[string]string)
	write := log.Record{
		Timestamp: time.Unix(0, 0),
		Value:     "Test log entry",
		Metadata:  meta,
	}
	meta["foo"] = "bar"
	k.Write(write)
	b := &bytes.Buffer{}
	w.Close()
	io.Copy(b, r)
	os.Stderr = s
	var read log.Record
	if err := json.Unmarshal(b.Bytes(), &read); err != nil {
		t.Fatalf("json.Unmarshal failed: %s", err.Error())
	}
	assert.Equal(t, write, read, "Write was not equal")

	assert.Nil(t, k.Read(), "Read should be unimplemented")

	stream, stop := k.Stream()
	records := []log.Record{}
	for s := range stream {
		records = append(records, s)
	}
	close(stop)
	assert.Equal(t, 0, len(records), "Stream should be unimplemented")

}

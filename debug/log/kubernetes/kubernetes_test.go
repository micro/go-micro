package kubernetes

import (
	"bytes"
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
	meta["foo"] = "bar"
	k.Write(log.Record{
		Timestamp: time.Unix(0, 0),
		Value:     "Test log entry",
		Metadata:  meta,
	})
	b := &bytes.Buffer{}
	w.Close()
	io.Copy(b, r)
	os.Stderr = s
	assert.Equal(t, "Test log entry", b.String(), "Write was not equal")

	assert.Nil(t, k.Read(), "Read should be unimplemented")

	stream := k.Stream(make(chan bool))
	records := []log.Record{}
	for s := range stream {
		records = append(records, s)
	}
	assert.Equal(t, 0, len(records), "Stream should be unimplemented")

}

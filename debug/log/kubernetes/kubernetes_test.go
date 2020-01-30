package kubernetes

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/debug/log"
	"github.com/stretchr/testify/assert"
)

func TestKubernetes(t *testing.T) {
	// TODO: fix local test running
	return

	if os.Getenv("IN_TRAVIS_CI") == "yes" {
		t.Skip("In Travis CI")
	}

	k := NewLog(log.Name("micro-network"))

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	s := os.Stderr
	os.Stderr = w
	meta := make(map[string]string)

	write := log.Record{
		Timestamp: time.Unix(0, 0).UTC(),
		Message:   "Test log entry",
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

	records, err := k.Read()
	assert.Nil(t, err, "Read should not error")
	assert.NotNil(t, records, "Read should return records")

	stream, err := k.Stream()
	if err != nil {
		t.Fatal(err)
	}

	records = nil

	go stream.Stop()

	for s := range stream.Chan() {
		records = append(records, s)
	}

	assert.Equal(t, 0, len(records), "Stream should return nothing")
}

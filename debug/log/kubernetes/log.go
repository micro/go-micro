package kubernetes

import "github.com/micro/go-micro/debug/log"

import (
	"encoding/json"
	"fmt"
	"os"
)

func write(l log.Record) error {
	m, err := json.Marshal(l)
	if err == nil {
		_, err := fmt.Fprintf(os.Stderr, "%s", m)
		return err
	}
	return err
}

type klogStreamer struct {
	streamChan chan log.Record
}

func (k *klogStreamer) Chan() <-chan log.Record {
	if k.streamChan == nil {
		k.streamChan = make(chan log.Record)
	}
	return k.streamChan
}

func (k *klogStreamer) Stop() error {
	close(k.streamChan)
	return nil
}

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
	// the k8s log stream
	streamChan chan log.Record
	// the stop chan
	stop chan bool
}

func (k *klogStreamer) Chan() <-chan log.Record {
	return k.streamChan
}

func (k *klogStreamer) Stop() error {
	select {
	case <-k.stop:
		return nil
	default:
		close(k.stop)
		close(k.streamChan)
	}
	return nil
}

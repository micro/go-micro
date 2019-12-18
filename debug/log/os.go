package log

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Should stream from OS
type osLog struct {
	sync.RWMutex
	subs map[string]*osStream
}

type osStream struct {
	stream chan Record
	stop   chan bool
}

// watch io stream
func (o *osLog) run() {
	// save outputs
	stdout := *os.Stdout
	stderr := *os.Stderr

	// new os pipe
	r, w := io.Pipe()

	// create new iopipes
	r1, w1, _ := os.Pipe()
	r2, w2, _ := os.Pipe()

	// create tea readers
	tee1 := io.TeeReader(r1, &stdout)
	tee2 := io.TeeReader(r2, &stderr)

	// start copying
	go io.Copy(w, tee1)
	go io.Copy(w, tee2)

	// set default go log output
	//log.SetOutput(w2)

	// replace os stdout and os stderr
	*os.Stdout = *w1
	*os.Stderr = *w2

	// this should short circuit everything
	defer func() {
		// reset stdout and stderr
		*os.Stdout = stdout
		*os.Stderr = stderr
		//log.SetOutput(stderr)

		// close all the outputs
		r.Close()
		r1.Close()
		r2.Close()
		w.Close()
		w1.Close()
		w2.Close()
	}()

	// read from standard error
	scanner := bufio.NewReader(r)

	for {
		// read the line
		line, err := scanner.ReadString('\n')
		if err != nil {
			return
		}
		// check if the line exists
		if len(line) == 0 {
			continue
		}
		// parse the record
		var r Record
		if line[0] == '{' {
			json.Unmarshal([]byte(line), &r)
		} else {
			r = Record{
				Timestamp: time.Now(),
				Value:     line,
				Metadata:  make(map[string]string),
			}
		}

		o.Lock()

		// bail if there's no subscribers
		if len(o.subs) == 0 {
			o.Unlock()
			return
		}

		// check subs and send to stream
		for id, sub := range o.subs {
			// send to stream
			select {
			case <-sub.stop:
				delete(o.subs, id)
			case sub.stream <- r:
				// send to stream
			default:
				// do not block
			}
		}

		o.Unlock()
	}
}

// Read reads log entries from the logger
func (o *osLog) Read(...ReadOption) ([]Record, error) {
	return []Record{}, nil
}

// Write writes records to log
func (o *osLog) Write(r Record) error {
	b, _ := json.Marshal(r)
	_, err := os.Stderr.Write(b)
	return err
}

// Stream log records
func (o *osLog) Stream() (Stream, error) {
	o.Lock()
	defer o.Unlock()

	// start stream watcher
	if len(o.subs) == 0 {
		go o.run()
	}

	// create stream
	st := &osStream{
		stream: make(chan Record, 128),
		stop:   make(chan bool),
	}

	// save stream
	o.subs[uuid.New().String()] = st

	return st, nil
}

func (o *osStream) Chan() <-chan Record {
	return o.stream
}

func (o *osStream) Stop() error {
	select {
	case <-o.stop:
		return nil
	default:
		close(o.stop)
	}
	return nil
}

func NewLog(opts ...Option) Log {
	return &osLog{
		subs: make(map[string]*osStream),
	}
}

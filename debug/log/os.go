package log

import (
	"bufio"
	"encoding/json"
	"os"
	"time"
)

// Should stream from OS
type osLog struct{}

type osStream struct {
	stream  chan Record
	scanner *bufio.Reader
	stop    chan bool
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
	// read from standard error
	scanner := bufio.NewReader(os.Stderr)
	stream := make(chan Record, 128)
	stop := make(chan bool)

	go func() {
		for {
			select {
			case <-stop:
				return
			default:
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
				// send to stream
				select {
				case <-stop:
					return
				case stream <- r:
				}
			}
		}
	}()

	return &osStream{
		stream:  stream,
		scanner: scanner,
		stop:    stop,
	}, nil
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
	return &osLog{}
}

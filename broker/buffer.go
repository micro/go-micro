package broker

import (
	"bytes"
	"io"
)

type closeBuffer struct {
	*bytes.Buffer
}

type writeBuffer struct {
	io.ReadCloser
}

func (b *closeBuffer) Close() error {
	b.Buffer.Reset()
	return nil
}

func (b *writeBuffer) Write([]byte) (int, error) {
	return 0, nil
}

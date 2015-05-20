package transport

import (
	"io"
)

type buffer struct {
	io.ReadWriter
}

func (b *buffer) Close() error {
	return nil
}

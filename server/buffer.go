package server

import (
	"io"
)

type buffer struct {
	io.Reader
	io.Writer
}

func (b *buffer) Close() error {
	return nil
}

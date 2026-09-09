package buf

import (
	"bytes"
)

type buffer struct {
	*bytes.Buffer
}

func (b *buffer) Close() error {
	b.Reset()
	return nil
}

func New(b *bytes.Buffer) *buffer {
	if b == nil {
		b = bytes.NewBuffer(nil)
	}
	return &buffer{b}
}

package buf

import (
	"bytes"
	"sync"
)

type pool struct {
	p sync.Pool
}

func (p *pool) Get() *bytes.Buffer {
	return p.p.Get().(*bytes.Buffer)
}

func (p *pool) Put(b *bytes.Buffer) {
	p.p.Put(b)
}

type buffer struct {
	*bytes.Buffer
}

func (b *buffer) Close() error {
	b.Buffer.Reset()
	return nil
}

func New(b *bytes.Buffer) *buffer {
	if b == nil {
		b = bytes.NewBuffer(nil)
	}
	return &buffer{b}
}

func NewPool() *pool {
	return &pool{
		p: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

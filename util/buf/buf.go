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
	b.Reset()
	p.p.Put(b)
}

type bytePool struct {
	p sync.Pool
}

func (p *bytePool) Get() []byte {
	return p.p.Get().([]byte)
}

func (p *bytePool) Put(b []byte) {
	for i, _ := range b {
		b[i] = '0'
	}
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

func NewBytePool(i int) *bytePool {
	return &bytePool{
		p: sync.Pool{
			New: func() interface{} {
				return make([]byte, i)
			},
		},
	}
}

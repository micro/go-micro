package transport

import "sync"

var http2BufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, DefaultBufSizeH2)
		return &buf
	},
}

func getHTTP2BufPool() *sync.Pool {
	return &http2BufPool
}

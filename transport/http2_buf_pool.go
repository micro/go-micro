package transport

import "sync"

var http2BufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, DefaultBufSizeH2)
	},
}

// getHTTP2BufPool returns the HTTP2 buffer pool.
func getHTTP2BufPool() *sync.Pool {
	return &http2BufPool
}

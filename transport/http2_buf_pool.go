package transport

import "sync"

var http2BufPool = sync.Pool{
       New: func() interface{} {
               return make([]byte, DefaultBufSizeH2)
       },
}

func getHTTP2BufPool() *sync.Pool {
       return &http2BufPool
}
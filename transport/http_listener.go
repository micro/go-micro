package transport

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"time"

	log "go-micro.dev/v5/logger"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type httpTransportListener struct {
	ht       *httpTransport
	listener net.Listener
}

func (h *httpTransportListener) Addr() string {
	return h.listener.Addr().String()
}

func (h *httpTransportListener) Close() error {
	return h.listener.Close()
}

func (h *httpTransportListener) Accept(fn func(Socket)) error {
	// Create handler mux
	// TODO: see if we should make a plugin out of the mux
	mux := http.NewServeMux()

	// Register our transport handler
	mux.HandleFunc("/", h.newHandler(fn))

	// Get optional handlers
	// TODO: This needs to be documented clearer, and examples provided
	if h.ht.opts.Context != nil {
		handlers, ok := h.ht.opts.Context.Value("http_handlers").(map[string]http.Handler)
		if ok {
			for pattern, handler := range handlers {
				mux.Handle(pattern, handler)
			}
		}
	}

	// Server ONLY supports HTTP1 + H2C
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: time.Second * 5,
	}

	// insecure connection use h2c
	if !(h.ht.opts.Secure || h.ht.opts.TLSConfig != nil) {
		srv.Handler = h2c.NewHandler(mux, &http2.Server{})
	}

	return srv.Serve(h.listener)
}

// newHandler creates a new HTTP transport handler passed to the mux.
func (h *httpTransportListener) newHandler(serveConn func(Socket)) func(rsp http.ResponseWriter, req *http.Request) {
	logger := h.ht.opts.Logger

	return func(rsp http.ResponseWriter, req *http.Request) {
		var (
			buf *bufio.ReadWriter
			con net.Conn
		)

		// HTTP1: read a regular request
		if req.ProtoMajor == 1 {
			b, err := io.ReadAll(req.Body)
			if err != nil {
				http.Error(rsp, err.Error(), http.StatusInternalServerError)
				return
			}

			req.Body = io.NopCloser(bytes.NewReader(b))

			// Hijack the conn
			// We also don't close the connection here, as it will be closed by
			// the httpTransportSocket
			hj, ok := rsp.(http.Hijacker)
			if !ok {
				// We're screwed
				http.Error(rsp, "cannot serve conn", http.StatusInternalServerError)
				return
			}

			conn, bufrw, err := hj.Hijack()
			if err != nil {
				http.Error(rsp, err.Error(), http.StatusInternalServerError)
				return
			}
			defer func() {
				if err := conn.Close(); err != nil {
					logger.Logf(log.ErrorLevel, "Failed to close TCP connection: %v", err)
				}
			}()

			buf = bufrw
			con = conn
		}

		// Buffered reader
		bufr := bufio.NewReader(req.Body)

		// Save the request
		ch := make(chan *http.Request, 1)
		ch <- req

		// Create a new transport socket
		sock := &httpTransportSocket{
			ht:     h.ht,
			w:      rsp,
			r:      req,
			rw:     buf,
			buf:    bufr,
			ch:     ch,
			conn:   con,
			local:  h.Addr(),
			remote: req.RemoteAddr,
			closed: make(chan bool),
		}

		// Execute the socket
		serveConn(sock)
	}
}

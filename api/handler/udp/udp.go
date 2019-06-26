// Package udp reads and write from a udp connection
package udp

import (
	"io"
	"net"
	"net/http"
)

type Handler struct{}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := net.Dial("udp", r.Host)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	go io.Copy(c, r.Body)
	// write response
	io.Copy(w, c)
}

func (h *Handler) String() string {
	return "udp"
}

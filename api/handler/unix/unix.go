// Package unix reads from a unix socket expecting it to be in /tmp/path
package unix

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
)

type Handler struct{}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sock := fmt.Sprintf("%s.sock", filepath.Clean(r.URL.Path))
	path := filepath.Join("/tmp", sock)

	c, err := net.Dial("unix", path)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	go io.Copy(c, r.Body)
	// write response
	io.Copy(w, c)
}

func (h *Handler) String() string {
	return "unix"
}

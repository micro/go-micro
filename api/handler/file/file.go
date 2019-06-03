// Package file serves file relative to the current directory
package file

import (
	"net/http"
)

type Handler struct{}

func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "."+r.URL.Path)
}

func (h *Handler) String() string {
	return "file"
}

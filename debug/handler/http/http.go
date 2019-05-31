// Package http provides http handlers
package http

import (
	"net/http"
)

type Handler struct{}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/debug/health":
	case "/debug/log":
	case "/debug/stats":
	case "/debug/trace":
	}
}

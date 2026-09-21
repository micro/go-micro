package mcp

import (
	"net/http"

	"go-micro.dev/v6/internal/browserorigin"
)

func (s *Server) checkOrigin(w http.ResponseWriter, r *http.Request) bool {
	return checkBrowserOrigin(w, r, s.opts.AllowedOrigins)
}

func checkBrowserOrigin(w http.ResponseWriter, r *http.Request, allowed []string) bool {
	w.Header().Add("Vary", "Origin")
	if !browserorigin.Allowed(r, allowed) {
		http.Error(w, "Forbidden origin", http.StatusForbidden)
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	return true
}

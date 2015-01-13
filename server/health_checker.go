package server

import (
	"io"
	"net/http"
	"net/url"
)

func registerHealthChecker(mux *http.ServeMux) {
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Path: HealthPath,
		},
	}
	if _, path := mux.Handler(req); path != HealthPath {
		mux.HandleFunc(HealthPath, func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "ok")
		})
	}
}

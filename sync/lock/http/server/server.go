// Package server implements the sync http server
package server

import (
	"net/http"

	"github.com/micro/go-micro/v2/sync/lock"
	lkhttp "github.com/micro/go-micro/v2/sync/lock/http"
)

func Handler(lk lock.Lock) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc(lkhttp.DefaultPath, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		id := r.Form.Get("id")
		if len(id) == 0 {
			return
		}

		switch r.Method {
		case "POST":
			err := lk.Acquire(id)
			if err != nil {
				http.Error(w, err.Error(), 500)
			}
		case "DELETE":
			err := lk.Release(id)
			if err != nil {
				http.Error(w, err.Error(), 500)
			}
		}
	})

	return mux
}

func Server(lk lock.Lock) *http.Server {
	server := &http.Server{
		Addr:    lkhttp.DefaultAddress,
		Handler: Handler(lk),
	}
	return server
}

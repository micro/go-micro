package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/registry"
)

// Write sets the status and body on a http ResponseWriter
func Write(w http.ResponseWriter, contentType string, status int, body string) {
	w.Header().Set("Content-Length", fmt.Sprintf("%v", len(body)))
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	fmt.Fprintf(w, `%v`, body)
}

// WriteBadRequestError sets a 400 status code
func WriteBadRequestError(w http.ResponseWriter, err error) {
	rawBody, err := json.Marshal(map[string]string{
		"error": err.Error(),
	})
	if err != nil {
		WriteInternalServerError(w, err)
		return
	}
	Write(w, "application/json", 400, string(rawBody))
}

// WriteInternalServerError sets a 500 status code
func WriteInternalServerError(w http.ResponseWriter, err error) {
	rawBody, err := json.Marshal(map[string]string{
		"error": err.Error(),
	})
	if err != nil {
		log.Println(err)
		return
	}
	Write(w, "application/json", 500, string(rawBody))
}

func NewRoundTripper(opts ...Option) http.RoundTripper {
	options := Options{
		Registry: registry.DefaultRegistry,
	}
	for _, o := range opts {
		o(&options)
	}

	return &roundTripper{
		rt:   http.DefaultTransport,
		st:   selector.Random,
		opts: options,
	}
}

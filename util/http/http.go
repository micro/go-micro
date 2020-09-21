package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/micro/go-micro/v3/metadata"
	"github.com/micro/go-micro/v3/router/registry"
	"github.com/micro/go-micro/v3/selector/random"
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
		Router: registry.NewRouter(),
	}
	for _, o := range opts {
		o(&options)
	}

	return &roundTripper{
		rt:   http.DefaultTransport,
		st:   random.NewSelector(),
		opts: options,
	}
}

// RequestToContext puts the `Authorization` header bearer token into context
// so calls to services will be authorized.
func RequestToContext(r *http.Request) context.Context {
	ctx := context.Background()
	md := make(metadata.Metadata)
	for k, v := range r.Header {
		md[k] = strings.Join(v, ",")
	}
	return metadata.NewContext(ctx, md)
}

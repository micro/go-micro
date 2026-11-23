package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
	log "go-micro.dev/v6/logger"
)

// Errors ...
var (
	ErrNoPodName = errors.New("no pod name provided")
	ErrNotFound  = errors.New("pod not found")
	ErrDecode    = errors.New("error decoding")
	ErrOther     = errors.New("unspecified error occurred in k8s registry")
)

// Response ...
type Response struct {
	res *http.Response
	err error
}

// Error returns an error.
func (r *Response) Error() error {
	return r.err
}

// StatusCode returns status code for response.
func (r *Response) StatusCode() int {
	return r.res.StatusCode
}

// Decode decodes body into `data`.
func (r *Response) Decode(data interface{}) error {
	if r.err != nil {
		return r.err
	}

	var err error
	defer func() {
		nerr := r.res.Body.Close()
		if err != nil {
			err = nerr
		}
	}()

	decoder := json.NewDecoder(r.res.Body)

	if err := decoder.Decode(&data); err != nil {
		return errors.Wrap(ErrDecode, err.Error())
	}

	return r.err
}

func newResponse(r *http.Response, err error) *Response {
	resp := &Response{
		res: r,
		err: err,
	}

	if err != nil {
		return resp
	}

	// Check if request is successful.
	s := resp.res.StatusCode
	if s == http.StatusOK || s == http.StatusCreated || s == http.StatusNoContent {
		return resp
	}

	// Decode returns early on an error response, so release the body here.
	defer func() { _ = resp.res.Body.Close() }()

	if resp.res.StatusCode == http.StatusNotFound {
		resp.err = ErrNotFound
		return resp
	}

	log.Errorf("K8s: request failed with code %v", resp.res.StatusCode)

	b, err := io.ReadAll(resp.res.Body)
	if err == nil {
		log.Errorf("K8s: request failed with body: %s", string(b))
	}

	resp.err = ErrOther

	return resp
}

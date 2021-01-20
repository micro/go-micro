package api

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	log "github.com/asim/go-micro/v3/logger"
)

// Errors ...
var (
	ErrNotFound = errors.New("K8s: not found")
	ErrDecode   = errors.New("K8s: error decoding")
	ErrOther    = errors.New("K8s: error")
)

// Status is an object that is returned when a request
// failed or delete succeeded.
// type Status struct {
// 	Kind    string `json:"kind"`
// 	Status  string `json:"status"`
// 	Message string `json:"message"`
// 	Reason  string `json:"reason"`
// 	Code    int    `json:"code"`
// }

// Response ...
type Response struct {
	res *http.Response
	err error

	body []byte
}

// Error returns an error
func (r *Response) Error() error {
	return r.err
}

// StatusCode returns status code for response
func (r *Response) StatusCode() int {
	return r.res.StatusCode
}

// Into decode body into `data`
func (r *Response) Into(data interface{}) error {
	if r.err != nil {
		return r.err
	}

	defer r.res.Body.Close()
	decoder := json.NewDecoder(r.res.Body)
	err := decoder.Decode(&data)
	if err != nil {
		return ErrDecode
	}

	return r.err
}

func newResponse(res *http.Response, err error) *Response {
	r := &Response{
		res: res,
		err: err,
	}

	if err != nil {
		return r
	}

	if r.res.StatusCode == http.StatusOK ||
		r.res.StatusCode == http.StatusCreated ||
		r.res.StatusCode == http.StatusNoContent {
		// Non error status code
		return r
	}

	if r.res.StatusCode == http.StatusNotFound {
		r.err = ErrNotFound
		return r
	}

	log.Errorf("K8s: request failed with code %v", r.res.StatusCode)

	b, err := ioutil.ReadAll(r.res.Body)
	if err == nil {
		log.Error("K8s: request failed with body:")
		log.Error(string(b))
	}
	r.err = ErrOther
	return r
}

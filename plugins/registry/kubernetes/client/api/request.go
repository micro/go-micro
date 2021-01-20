package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/asim/go-micro/plugins/registry/kubernetes/v3/client/watch"
)

// Request is used to construct a http request for the k8s API.
type Request struct {
	client    *http.Client
	header    http.Header
	params    url.Values
	method    string
	host      string
	namespace string

	resource     string
	resourceName *string
	body         io.Reader

	err error
}

// Params is the object to pass in to set parameters
// on a request.
type Params struct {
	LabelSelector map[string]string
	Watch         bool
}

// verb sets method
func (r *Request) verb(method string) *Request {
	r.method = method
	return r
}

// Get request
func (r *Request) Get() *Request {
	return r.verb("GET")
}

// Post request
func (r *Request) Post() *Request {
	return r.verb("POST")
}

// Put request
func (r *Request) Put() *Request {
	return r.verb("PUT")
}

// Patch request
// https://github.com/kubernetes/kubernetes/blob/master/docs/devel/api-conventions.md#patch-operations
func (r *Request) Patch() *Request {
	return r.verb("PATCH").SetHeader("Content-Type", "application/strategic-merge-patch+json")
}

// Delete request
func (r *Request) Delete() *Request {
	return r.verb("DELETE")
}

// Namespace is to set the namespace to operate on
func (r *Request) Namespace(s string) *Request {
	r.namespace = s
	return r
}

// Resource is the type of resource the operation is
// for, such as "services", "endpoints" or "pods"
func (r *Request) Resource(s string) *Request {
	r.resource = s
	return r
}

// Name is for targeting a specific resource by id
func (r *Request) Name(s string) *Request {
	r.resourceName = &s
	return r
}

// Body pass in a body to set, this is for POST, PUT
// and PATCH requests
func (r *Request) Body(in interface{}) *Request {
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(&in); err != nil {
		r.err = err
		return r
	}
	r.body = b
	return r
}

// Params isused to set parameters on a request
func (r *Request) Params(p *Params) *Request {
	for k, v := range p.LabelSelector {
		// create new key=value pair
		value := fmt.Sprintf("%s=%s", k, v)
		// check if there's an existing value
		if label := r.params.Get("labelSelector"); len(label) > 0 {
			value = fmt.Sprintf("%s,%s", label, value)
		}
		// set and overwrite the value
		r.params.Set("labelSelector", value)
	}

	return r
}

// SetHeader sets a header on a request with
// a `key` and `value`
func (r *Request) SetHeader(key, value string) *Request {
	r.header.Add(key, value)
	return r
}

// request builds the http.Request from the options
func (r *Request) request() (*http.Request, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/%s/", r.host, r.namespace, r.resource)

	// append resourceName if it is present
	if r.resourceName != nil {
		url += *r.resourceName
	}

	// append any query params
	if len(r.params) > 0 {
		url += "?" + r.params.Encode()
	}

	// build request
	req, err := http.NewRequest(r.method, url, r.body)
	if err != nil {
		return nil, err
	}

	// set headers on request
	req.Header = r.header
	return req, nil
}

// Do builds and triggers the request
func (r *Request) Do() *Response {
	if r.err != nil {
		return &Response{
			err: r.err,
		}
	}

	req, err := r.request()
	if err != nil {
		return &Response{
			err: err,
		}
	}

	res, err := r.client.Do(req)
	if err != nil {
		return &Response{
			err: err,
		}
	}

	// return res, err
	return newResponse(res, err)
}

// Watch builds and triggers the request, but
// will watch instead of return an object
func (r *Request) Watch() (watch.Watch, error) {
	if r.err != nil {
		return nil, r.err
	}

	r.params.Set("watch", "true")

	req, err := r.request()
	if err != nil {
		return nil, err
	}

	w, err := watch.NewBodyWatcher(req, r.client)
	return w, err
}

// Options ...
type Options struct {
	Host        string
	Namespace   string
	BearerToken *string
	Client      *http.Client
}

// NewRequest creates a k8s api request
func NewRequest(opts *Options) *Request {
	req := &Request{
		header:    make(http.Header),
		params:    make(url.Values),
		client:    opts.Client,
		namespace: opts.Namespace,
		host:      opts.Host,
	}

	if opts.BearerToken != nil {
		req.SetHeader("Authorization", "Bearer "+*opts.BearerToken)
	}

	return req
}

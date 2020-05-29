package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/micro/go-micro/v2/logger"
)

// Request is used to construct a http request for the k8s API.
type Request struct {
	// the request context
	context   context.Context
	client    *http.Client
	header    http.Header
	params    url.Values
	method    string
	host      string
	namespace string

	resource     string
	resourceName *string
	subResource  *string
	body         io.Reader

	err error
}

// Params is the object to pass in to set parameters
// on a request.
type Params struct {
	LabelSelector map[string]string
	Annotations   map[string]string
	Additional    map[string]string
}

// verb sets method
func (r *Request) verb(method string) *Request {
	r.method = method
	return r
}

func (r *Request) Context(ctx context.Context) {
	r.context = ctx
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
func (r *Request) Patch() *Request {
	return r.verb("PATCH")
}

// Delete request
func (r *Request) Delete() *Request {
	return r.verb("DELETE")
}

// Namespace is to set the namespace to operate on
func (r *Request) Namespace(s string) *Request {
	if len(s) > 0 {
		r.namespace = s
	}
	return r
}

// Resource is the type of resource the operation is
// for, such as "services", "endpoints" or "pods"
func (r *Request) Resource(s string) *Request {
	r.resource = s
	return r
}

// SubResource sets a subresource on a resource,
// e.g. pods/log for pod logs
func (r *Request) SubResource(s string) *Request {
	r.subResource = &s
	return r
}

// Name is for targeting a specific resource by id
func (r *Request) Name(s string) *Request {
	r.resourceName = &s
	return r
}

// Body pass in a body to set, this is for POST, PUT and PATCH requests
func (r *Request) Body(in interface{}) *Request {
	b := new(bytes.Buffer)
	// if we're not sending YAML request, we encode to JSON
	if r.header.Get("Content-Type") != "application/yaml" {
		if err := json.NewEncoder(b).Encode(&in); err != nil {
			r.err = err
			return r
		}
		r.body = b
		return r
	}

	// if application/yaml is set, we assume we get a raw bytes so we just copy over
	body, ok := in.(io.Reader)
	if !ok {
		r.err = errors.New("invalid data")
		return r
	}
	// copy over data to the bytes buffer
	if _, err := io.Copy(b, body); err != nil {
		r.err = err
		return r
	}

	r.body = b
	return r
}

// Params isused to set paramters on a request
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
	for k, v := range p.Additional {
		r.params.Set(k, v)
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
	var url string
	switch r.resource {
	case "namespace":
		// /api/v1/namespaces/
		url = fmt.Sprintf("%s/api/v1/namespaces/", r.host)
	case "deployment":
		// /apis/apps/v1/namespaces/{namespace}/deployments/{name}
		url = fmt.Sprintf("%s/apis/apps/v1/namespaces/%s/%ss/", r.host, r.namespace, r.resource)
	default:
		// /api/v1/namespaces/{namespace}/{resource}
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/%ss/", r.host, r.namespace, r.resource)
	}

	// append resourceName if it is present
	if r.resourceName != nil {
		url += *r.resourceName
		if r.subResource != nil {
			url += "/" + *r.subResource
		}
	}

	// append any query params
	if len(r.params) > 0 {
		url += "?" + r.params.Encode()
	}

	var req *http.Request
	var err error

	// build request
	if r.context != nil {
		req, err = http.NewRequestWithContext(r.context, r.method, url, r.body)
	} else {
		req, err = http.NewRequest(r.method, url, r.body)
	}
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

	logger.Debugf("[Kubernetes] %v %v", req.Method, req.URL.String())
	res, err := r.client.Do(req)
	if err != nil {
		return &Response{
			err: err,
		}
	}

	// return res, err
	return newResponse(res, err)
}

// Raw performs a Raw HTTP request to the Kubernetes API
func (r *Request) Raw() (*http.Response, error) {
	req, err := r.request()
	if err != nil {
		return nil, err
	}

	res, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
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

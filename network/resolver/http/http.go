// Package http resolves names to network addresses using a http request
package http

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/micro/go-micro/v2/network/resolver"
)

// Resolver is a HTTP network resolver
type Resolver struct {
	// If not set, defaults to http
	Proto string

	// Path sets the path to lookup. Defaults to /network
	Path string

	// Host url to use for the query
	Host string
}

type Response struct {
	Nodes []*resolver.Record `json:"nodes,omitempty"`
}

// Resolve assumes ID is a domain which can be converted to a http://name/network request
func (r *Resolver) Resolve(name string) ([]*resolver.Record, error) {
	proto := "https"
	host := "go.micro.mu"
	path := "/network/nodes"

	if len(r.Proto) > 0 {
		proto = r.Proto
	}

	if len(r.Path) > 0 {
		path = r.Path
	}

	if len(r.Host) > 0 {
		host = r.Host
	}

	uri := &url.URL{
		Scheme: proto,
		Path:   path,
		Host:   host,
	}
	q := uri.Query()
	q.Set("name", name)
	uri.RawQuery = q.Encode()

	rsp, err := http.Get(uri.String())
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return nil, errors.New("non 200 response")
	}
	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	// encoding format is assumed to be json
	var response *Response

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, err
	}

	return response.Nodes, nil
}

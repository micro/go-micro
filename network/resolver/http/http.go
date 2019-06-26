// Package http resolves ids to network addresses using a http request
package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/micro/go-micro/network/resolver"
)

type Resolver struct {
	// If not set, defaults to http
	Proto string

	// Path sets the path to lookup. Defaults to /network
	Path string
}

// Resolve assumes ID is a domain which can be converted to a http://id/network request
func (r *Resolver) Resolve(id string) ([]*resolver.Record, error) {
	proto := "http"
	path := "/network"

	if len(r.Proto) > 0 {
		proto = r.Proto
	}

	if len(r.Path) > 0 {
		path = r.Path
	}

	uri := &url.URL{
		Scheme: proto,
		Path:   path,
		Host:   id,
	}

	rsp, err := http.Get(uri.String())
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	// encoding format is assumed to be json
	var records []*resolver.Record

	if err := json.Unmarshal(b, &records); err != nil {
		return nil, err
	}

	return records, nil
}

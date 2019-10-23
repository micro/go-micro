// Package http adds a http lock implementation
package http

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/micro/go-micro/sync/lock"
)

var (
	DefaultPath    = "/sync/lock"
	DefaultAddress = "localhost:8080"
)

type httpLock struct {
	opts lock.Options
}

func (h *httpLock) url(do, id string) (string, error) {
	sum := crc32.ChecksumIEEE([]byte(id))
	node := h.opts.Nodes[sum%uint32(len(h.opts.Nodes))]

	// parse the host:port or whatever
	uri, err := url.Parse(node)
	if err != nil {
		return "", err
	}

	if len(uri.Scheme) == 0 {
		uri.Scheme = "http"
	}

	// set path
	// build path
	path := filepath.Join(DefaultPath, do, h.opts.Prefix, id)
	uri.Path = path

	// return url
	return uri.String(), nil
}

func (h *httpLock) Acquire(id string, opts ...lock.AcquireOption) error {
	var options lock.AcquireOptions
	for _, o := range opts {
		o(&options)
	}

	uri, err := h.url("acquire", id)
	if err != nil {
		return err
	}

	ttl := fmt.Sprintf("%d", int64(options.TTL.Seconds()))
	wait := fmt.Sprintf("%d", int64(options.Wait.Seconds()))

	rsp, err := http.PostForm(uri, url.Values{
		"id":   {id},
		"ttl":  {ttl},
		"wait": {wait},
	})
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	// success
	if rsp.StatusCode == 200 {
		return nil
	}

	// return error
	return errors.New(string(b))
}

func (h *httpLock) Release(id string) error {
	uri, err := h.url("release", id)
	if err != nil {
		return err
	}

	vals := url.Values{
		"id": {id},
	}

	req, err := http.NewRequest("DELETE", uri, strings.NewReader(vals.Encode()))
	if err != nil {
		return err
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	// success
	if rsp.StatusCode == 200 {
		return nil
	}

	// return error
	return errors.New(string(b))
}

func NewLock(opts ...lock.Option) lock.Lock {
	var options lock.Options
	for _, o := range opts {
		o(&options)
	}

	if len(options.Nodes) == 0 {
		options.Nodes = []string{DefaultAddress}
	}

	return &httpLock{
		opts: options,
	}
}

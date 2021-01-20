// Package url loads changesets from a url
package url

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/micro/go-micro/v2/config/source"
)

type urlSource struct {
	url  string
	opts source.Options
}

var (
	DefaultURL = "http://localhost:8080/config"
)

func (u *urlSource) Read() (*source.ChangeSet, error) {
	rsp, err := http.Get(u.url)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	ft := format(rsp.Header.Get("Content-Type"))
	if len(ft) == 0 {
		ft = u.opts.Encoder.String()
	}

	cs := &source.ChangeSet{
		Data:      b,
		Format:    ft,
		Timestamp: time.Now(),
		Source:    u.String(),
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (u *urlSource) Watch() (source.Watcher, error) {
	return newWatcher(u)
}

// Write is unsupported
func (u *urlSource) Write(cs *source.ChangeSet) error {
	return nil
}

func (u *urlSource) String() string {
	return "url"
}

func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)

	url, ok := options.Context.Value(urlKey{}).(string)
	if !ok {
		url = DefaultURL
	}

	return &urlSource{url: url, opts: options}
}

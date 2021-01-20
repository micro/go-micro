// Package proxy is a registry plugin for the micro proxy
package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/registry"
)

type proxy struct {
	opts registry.Options
}

func init() {
	cmd.DefaultRegistries["proxy"] = NewRegistry
}

func configure(s *proxy, opts ...registry.Option) error {
	for _, o := range opts {
		o(&s.opts)
	}
	var addrs []string
	for _, addr := range s.opts.Addrs {
		if len(addr) > 0 {
			addrs = append(addrs, addr)
		}
	}
	if len(addrs) == 0 {
		addrs = []string{"localhost:8081"}
	}
	registry.Addrs(addrs...)(&s.opts)
	return nil
}

func newRegistry(opts ...registry.Option) registry.Registry {
	s := &proxy{
		opts: registry.Options{},
	}
	configure(s, opts...)
	return s
}

func (s *proxy) Init(opts ...registry.Option) error {
	return configure(s, opts...)
}

func (s *proxy) Options() registry.Options {
	return s.opts
}

func (s *proxy) Register(service *registry.Service, opts ...registry.RegisterOption) error {
	b, err := json.Marshal(service)
	if err != nil {
		return err
	}

	var gerr error
	for _, addr := range s.opts.Addrs {
		scheme := "http"
		if s.opts.Secure {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s/registry", scheme, addr)
		rsp, err := http.Post(url, "application/json", bytes.NewReader(b))
		if err != nil {
			gerr = err
			continue
		}
		if rsp.StatusCode != 200 {
			b, err := ioutil.ReadAll(rsp.Body)
			if err != nil {
				return err
			}
			rsp.Body.Close()
			gerr = errors.New(string(b))
			continue
		}
		io.Copy(ioutil.Discard, rsp.Body)
		rsp.Body.Close()
		return nil
	}
	return gerr
}

func (s *proxy) Deregister(service *registry.Service, opts ...registry.DeregisterOption) error {
	b, err := json.Marshal(service)
	if err != nil {
		return err
	}

	var gerr error
	for _, addr := range s.opts.Addrs {
		scheme := "http"
		if s.opts.Secure {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s/registry", scheme, addr)

		req, err := http.NewRequest("DELETE", url, bytes.NewReader(b))
		if err != nil {
			gerr = err
			continue
		}

		rsp, err := http.DefaultClient.Do(req)
		if err != nil {
			gerr = err
			continue
		}

		if rsp.StatusCode != 200 {
			b, err := ioutil.ReadAll(rsp.Body)
			if err != nil {
				return err
			}
			rsp.Body.Close()
			gerr = errors.New(string(b))
			continue
		}

		io.Copy(ioutil.Discard, rsp.Body)
		rsp.Body.Close()
		return nil
	}
	return gerr
}

func (s *proxy) GetService(service string, opts ...registry.GetOption) ([]*registry.Service, error) {
	var gerr error
	for _, addr := range s.opts.Addrs {
		scheme := "http"
		if s.opts.Secure {
			scheme = "https"
		}

		url := fmt.Sprintf("%s://%s/registry?service=%s", scheme, addr, url.QueryEscape(service))
		rsp, err := http.Get(url)
		if err != nil {
			gerr = err
			continue
		}

		if rsp.StatusCode != 200 {
			b, err := ioutil.ReadAll(rsp.Body)
			if err != nil {
				return nil, err
			}
			rsp.Body.Close()
			gerr = errors.New(string(b))
			continue
		}

		b, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			gerr = err
			continue
		}
		rsp.Body.Close()
		var services []*registry.Service
		if err := json.Unmarshal(b, &services); err != nil {
			gerr = err
			continue
		}
		return services, nil
	}
	return nil, gerr
}

func (s *proxy) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	var gerr error
	for _, addr := range s.opts.Addrs {
		scheme := "http"
		if s.opts.Secure {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s/registry", scheme, addr)
		rsp, err := http.Get(url)
		if err != nil {
			gerr = err
			continue
		}

		if rsp.StatusCode != 200 {
			b, err := ioutil.ReadAll(rsp.Body)
			if err != nil {
				return nil, err
			}
			rsp.Body.Close()
			gerr = errors.New(string(b))
			continue
		}

		b, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			gerr = err
			continue
		}
		rsp.Body.Close()
		var services []*registry.Service
		if err := json.Unmarshal(b, &services); err != nil {
			gerr = err
			continue
		}
		return services, nil
	}
	return nil, gerr
}

func (s *proxy) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	watch := func(addr string) (registry.Watcher, error) {
		scheme := "ws"
		if s.opts.Secure {
			scheme = "wss"
		}
		url := fmt.Sprintf("%s://%s/registry", scheme, addr)
		// service filter
		if len(wo.Service) > 0 {
			url = url + "?service=" + wo.Service
		}
		return newWatcher(url)
	}

	var gerr error
	for _, addr := range s.opts.Addrs {
		w, err := watch(addr)
		if err != nil {
			gerr = err
			continue
		}
		return w, nil
	}
	return nil, gerr
}

func (s *proxy) String() string {
	return "proxy"
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	return newRegistry(opts...)
}

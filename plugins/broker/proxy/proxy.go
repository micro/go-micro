// Package proxy is a broker using the micro proxy
package proxy

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/cmd"
)

type sidecar struct {
	opts broker.Options
}

func init() {
	cmd.DefaultBrokers["sidecar"] = NewBroker
}

func newBroker(opts ...broker.Option) broker.Broker {
	var options broker.Options
	for _, o := range opts {
		o(&options)
	}

	var addrs []string

	for _, addr := range options.Addrs {
		if len(addr) == 0 {
			continue
		}
		addrs = append(addrs, addr)
	}

	if len(addrs) == 0 {
		addrs = []string{"localhost:8081"}
	}

	broker.Addrs(addrs...)(&options)

	return &sidecar{
		opts: options,
	}
}

func (s *sidecar) Options() broker.Options {
	return s.opts
}

func (s *sidecar) Address() string {
	if len(s.opts.Addrs) == 0 {
		return "localhost:8081"
	}
	return s.opts.Addrs[0]
}

func (s *sidecar) Connect() error {
	return nil
}

func (s *sidecar) Disconnect() error {
	return nil
}

func (s *sidecar) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&s.opts)
	}

	addrs := make([]string, 0, len(s.opts.Addrs))
	for _, addr := range s.opts.Addrs {
		if len(addr) == 0 {
			continue
		}
		addrs = append(addrs, addr)
	}

	if len(addrs) == 0 {
		addrs = []string{"localhost:8081"}
	}

	broker.Addrs(addrs...)(&s.opts)
	return nil
}

func (s *sidecar) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	pub := func(addr string) error {
		scheme := "http"
		if s.opts.Secure {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s/broker?topic=%s", scheme, addr, topic)

		req, err := http.NewRequest("POST", url, bytes.NewReader(msg.Body))
		if err != nil {
			return err
		}

		for k, v := range msg.Header {
			req.Header.Set(k, v)
		}

		rsp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		// discard response
		io.Copy(ioutil.Discard, rsp.Body)
		rsp.Body.Close()

		return nil
	}

	var gerr error
	for _, addr := range s.opts.Addrs {
		if err := pub(addr); err != nil {
			gerr = err
			continue
		}
		return nil
	}
	return gerr
}

func (s *sidecar) Subscribe(topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	var options broker.SubscribeOptions
	for _, o := range opts {
		o(&options)
	}

	sub := func(addr string) (broker.Subscriber, error) {
		scheme := "ws"
		if s.opts.Secure {
			scheme = "wss"
		}
		url := fmt.Sprintf("%s://%s/broker?topic=%s", scheme, addr, topic)
		if len(options.Queue) > 0 {
			url = fmt.Sprintf("%s&queue=%s", url, options.Queue)
		}
		return newSubscriber(url, topic, h, options)
	}

	var gerr error
	for _, addr := range s.opts.Addrs {
		s, err := sub(addr)
		if err != nil {
			gerr = err
			continue
		}
		return s, nil
	}
	return nil, gerr
}

func (s *sidecar) String() string {
	return "sidecar"
}

func NewBroker(opts ...broker.Option) broker.Broker {
	return newBroker(opts...)
}

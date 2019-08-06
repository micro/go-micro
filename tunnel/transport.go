package tunnel

import "github.com/micro/go-micro/transport"

type tunTransport struct {
	options transport.Options
}

type tunClient struct {
	*tunSocket
	options transport.DialOptions
}

type tunListener struct {
	conn chan *tunSocket
}

func newTransport(opts ...transport.Option) transport.Transport {
	var options transport.Options

	for _, o := range opts {
		o(&options)
	}

	return &tunTransport{
		options: options,
	}
}

func (t *tunTransport) Init(opts ...transport.Option) error {
	for _, o := range opts {
		o(&t.options)
	}
	return nil
}

func (t *tunTransport) Options() transport.Options {
	return t.options
}

func (t *tunTransport) Dial(addr string, opts ...transport.DialOption) (transport.Client, error) {
	return nil, nil
}

func (t *tunTransport) Listen(addr string, opts ...transport.ListenOption) (transport.Listener, error) {
	return nil, nil
}

func (t *tunTransport) String() string {
	return "micro"
}

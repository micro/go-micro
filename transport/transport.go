package transport

type Message struct {
	Header map[string]string
	Body   []byte
}

type Socket interface {
	Recv(*Message) error
	Send(*Message) error
	Close() error
}

type Client interface {
	Recv(*Message) error
	Send(*Message) error
	Close() error
}

type Listener interface {
	Addr() string
	Close() error
	Accept(func(Socket)) error
}

type Transport interface {
	Dial(addr string, opts ...DialOption) (Client, error)
	Listen(addr string) (Listener, error)
	String() string
}

type Options struct{}

type DialOptions struct {
	stream bool
}

type Option func(*Options)

type DialOption func(*DialOptions)

var (
	DefaultTransport Transport = newHttpTransport([]string{})
)

func WithStream() DialOption {
	return func(o *DialOptions) {
		o.stream = true
	}
}

func NewTransport(addrs []string, opt ...Option) Transport {
	return newHttpTransport(addrs, opt...)
}

func Dial(addr string, opts ...DialOption) (Client, error) {
	return DefaultTransport.Dial(addr, opts...)
}

func Listen(addr string) (Listener, error) {
	return DefaultTransport.Listen(addr)
}

func String() string {
	return DefaultTransport.String()
}

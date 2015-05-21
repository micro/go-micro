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
	Send(*Message) (*Message, error)
	Close() error
}

type Listener interface {
	Addr() string
	Close() error
	Accept(func(Socket)) error
}

type Transport interface {
	Dial(addr string) (Client, error)
	Listen(addr string) (Listener, error)
}

var (
	DefaultTransport Transport = NewHttpTransport([]string{})
)

func Dial(addr string) (Client, error) {
	return DefaultTransport.Dial(addr)
}

func Listen(addr string) (Listener, error) {
	return DefaultTransport.Listen(addr)
}

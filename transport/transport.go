package transport

type Message struct {
	Header map[string]string
	Body   []byte
}

type Socket interface {
	Recv() (*Message, error)
	WriteHeader(string, string)
	Write([]byte) error
}

type Client interface {
	Send(*Message) (*Message, error)
	Close() error
}

type Server interface {
	Addr() string
	Close() error
	Serve(func(Socket)) error
}

type Transport interface {
	NewClient(name, addr string) (Client, error)
	NewServer(name, addr string) (Server, error)
}

var (
	DefaultTransport Transport = NewHttpTransport([]string{})
)

func NewClient(name, addr string) (Client, error) {
	return DefaultTransport.NewClient(name, addr)
}

func NewServer(name, addr string) (Server, error) {
	return DefaultTransport.NewServer(name, addr)
}

package broker

type Broker interface {
	Address() string
	Connect() error
	Disconnect() error
	Init() error
	Publish(string, *Message) error
	Subscribe(string, Handler) (Subscriber, error)
}

type Handler func(*Message)

type Message struct {
	Header map[string]string
	Body   []byte
}

type Subscriber interface {
	Topic() string
	Unsubscribe() error
}

type options struct{}

type Option func(*options)

var (
	DefaultBroker Broker = newHttpBroker([]string{})
)

func NewBroker(addrs []string, opt ...Option) Broker {
	return newHttpBroker(addrs, opt...)
}

func Init() error {
	return DefaultBroker.Init()
}

func Connect() error {
	return DefaultBroker.Connect()
}

func Disconnect() error {
	return DefaultBroker.Disconnect()
}

func Publish(topic string, msg *Message) error {
	return DefaultBroker.Publish(topic, msg)
}

func Subscribe(topic string, handler Handler) (Subscriber, error) {
	return DefaultBroker.Subscribe(topic, handler)
}

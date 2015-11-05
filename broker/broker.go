package broker

// Broker is the top level interface of this eponymous package.
// It abstract away the underlying communication mechanism and enables
// publish and subscribe operations with a given system
type Broker interface {
	Address() string
	Connect() error
	Disconnect() error
	Init() error
	Publish(string, *Message) error
	NewSubscriber(string, string) (Subscriber, error)
}

// HandlerFunc represents the typical signature of a message handler function
type HandlerFunc func(*Message) error

// Handler denotes any type which is able to handle a brokered message and acknowledge it
type Handler interface {
	Handle(*Message) error
	Ack(*Message) error
}

// Message is a message sent or received via the broker
type Message struct {
	Header map[string]string
	Body   []byte
}

// Subscriber abstracts away subscription mechanisms a presents a consistent way to receive data
type Subscriber interface {
	Topic() string
	Name() string
	SetHandlerFunc(HandlerFunc, int)
	Subscribe() error
	Unsubscribe() error
}

type options struct{}

// Option refers to anyfunction that can take a pointer to options
type Option func(*options)

var (
	// DefaultBroker represents a system wide broker by default
	DefaultBroker = newHTTPBroker([]string{})
)

// NewBroker creates and returns a default broker instance
func NewBroker(addrs []string, opt ...Option) Broker {
	return newHTTPBroker(addrs, opt...)
}

// Init initialises the default broker instance
func Init() error {
	return DefaultBroker.Init()
}

// Connect initiates the connection of the default broker instance
func Connect() error {
	return DefaultBroker.Connect()
}

// Disconnect disconnects the default broker instance
func Disconnect() error {
	return DefaultBroker.Disconnect()
}

// Publish publish a message using the default broker instance
func Publish(topic string, msg *Message) error {
	return DefaultBroker.Publish(topic, msg)
}

// Subscribe creates a subscriber and initiates the subscription
func Subscribe(topic, name string, handlerFunc func(*Message) error) (Subscriber, error) {
	subscriber, err := DefaultBroker.NewSubscriber(name, topic)

	if err != nil {
		return nil, err
	}

	return subscriber, subscriber.Subscribe()
}

package registry

const (
	MessageEvent EventType = iota
	CreateEvent
	DeleteEvent
	UpdateEvent
)

// Watcher is an interface that returns updates
// about services within the registry.
type Watcher interface {
	// Next is a blocking call
	Next() (*Event, error)
	// Chan returns an event channel
	Chan() (<-chan *Event, error)
	// Stop stops all events
	Stop()
}

// Event is returned by a call to Next on
// the watcher. Types can be create, update, delete, ...
type Event struct {
	// type of event e.g create, update, delete, expire, ...
	Type EventType
	// service for which this occured
	Service *Service
	// the event message
	Message *Message
}

// Message is a message received as part of an event
type Message struct {
	// metadata associated with the message
	Header map[string]string
	// opaque body for an encoded message
	Body []byte
}

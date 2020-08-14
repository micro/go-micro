package events

import (
	"encoding/json"
	"time"
)

// Stream of events
type Stream interface {
	Publish(topic string, opts ...PublishOption) error
	Subscribe(topic string, opts ...SubscribeOption) (<-chan Event, error)
}

// Store of events
type Store interface {
	Read(opts ...ReadOption) ([]*Event, error)
	Write(event *Event, opts ...WriteOption) error
}

// Event is the object returned by the broker when you subscribe to a topic
type Event struct {
	// ID to uniquely identify the event
	ID string
	// Topic the event was published on
	Topic string
	// Timestamp of the event
	Timestamp time.Time
	// Metadata contains the encoded event was indexed by
	Metadata map[string]string
	// Payload contains the json encoded payload
	Payload []byte
}

// Unmarshal the events payload into an object
func (e *Event) Unmarshal(v interface{}) error {
	return json.Unmarshal(e.Payload, v)
}

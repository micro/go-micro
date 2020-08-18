// Package events contains interfaces for managing events within distributed systems
package events

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrMissingTopic is returned if a blank topic was provided to publish
	ErrMissingTopic = errors.New("Missing topic")
	// ErrEncodingMessage is returned from publish if there was an error encoding the message option
	ErrEncodingMessage = errors.New("Error encoding message")
)

// Stream of events
type Stream interface {
	Publish(topic string, opts ...PublishOption) error
	Subscribe(opts ...SubscribeOption) (<-chan Event, error)
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
	// Topic of event, e.g. "registry.service.created"
	Topic string
	// Timestamp of the event
	Timestamp time.Time
	// Metadata contains the encoded event was indexed by
	Metadata map[string]string
	// Payload contains the encoded message
	Payload []byte
}

// Unmarshal the events message into an object
func (e *Event) Unmarshal(v interface{}) error {
	return json.Unmarshal(e.Payload, v)
}

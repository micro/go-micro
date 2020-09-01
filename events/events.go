// Package events is for event streaming and storage
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

// Stream is an event streaming interface
type Stream interface {
	Publish(topic string, msg interface{}, opts ...PublishOption) error
	Subscribe(topic string, opts ...SubscribeOption) (<-chan Event, error)
}

// Store is an event store interface
type Store interface {
	Read(topic string, opts ...ReadOption) ([]*Event, error)
	Write(event *Event, opts ...WriteOption) error
}

type AckFunc func() error
type NackFunc func() error

// Event is the object returned by the broker when you subscribe to a topic
type Event struct {
	// ID to uniquely identify the event
	ID string
	// Topic of event, e.g. "registry.service.created"
	Topic string
	// Timestamp of the event
	Timestamp time.Time
	// Metadata contains the values the event was indexed by
	Metadata map[string]string
	// Payload contains the encoded message
	Payload []byte

	ackFunc  AckFunc
	nackFunc NackFunc
}

// Unmarshal the events message into an object
func (e *Event) Unmarshal(v interface{}) error {
	return json.Unmarshal(e.Payload, v)
}

func (e *Event) Ack() error {
	return e.ackFunc()
}

func (e *Event) SetAckFunc(f AckFunc) {
	e.ackFunc = f
}

func (e *Event) Nack() error {
	return e.nackFunc()
}

func (e *Event) SetNackFunc(f NackFunc) {
	e.nackFunc = f
}

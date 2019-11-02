package watch

import "encoding/json"

// Watch ...
type Watch interface {
	Stop()
	ResultChan() <-chan Event
}

// EventType defines the possible types of events.
type EventType string

// EventTypes used
const (
	Added    EventType = "ADDED"
	Modified EventType = "MODIFIED"
	Deleted  EventType = "DELETED"
	Error    EventType = "ERROR"
)

// Event represents a single event to a watched resource.
type Event struct {
	Type   EventType       `json:"type"`
	Object json.RawMessage `json:"object"`
}

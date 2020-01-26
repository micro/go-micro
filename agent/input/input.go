// Package input is an interface for bot inputs
package input

import (
	"github.com/micro/cli/v2"
)

type EventType string

const (
	TextEvent EventType = "text"
)

var (
	// Inputs keyed by name
	// Example slack or hipchat
	Inputs = map[string]Input{}
)

// Event is the unit sent and received
type Event struct {
	Type EventType
	From string
	To   string
	Data []byte
	Meta map[string]interface{}
}

// Input is an interface for sources which
// provide a way to communicate with the bot.
// Slack, HipChat, XMPP, etc.
type Input interface {
	// Provide cli flags
	Flags() []cli.Flag
	// Initialise input using cli context
	Init(*cli.Context) error
	// Stream events from the input
	Stream() (Conn, error)
	// Start the input
	Start() error
	// Stop the input
	Stop() error
	// name of the input
	String() string
}

// Conn interface provides a way to
// send and receive events. Send and
// Recv both block until succeeding
// or failing.
type Conn interface {
	Close() error
	Recv(*Event) error
	Send(*Event) error
}

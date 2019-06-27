// Package peer is used for peer to peer communication
package peer

// Peer is an interface for peer to peer communication. 
// It builds on the Client and Server interfaces to provide 
// Peer based communication for instances of a service.
type Peer interface {
	// Send to a peer
	Send(*Message) error
	// Accept from a peer
	Accept(*Message) error
	// Broadcast to all peers
	Broadcast(*Message) error
}


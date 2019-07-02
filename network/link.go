package network

import (
	"sync"

	pb "github.com/micro/go-micro/network/proto"
)

type link struct {
	// the embedded node
	*node

	// the link id
	id string

	// queue buffer for this link
	queue chan *Message

	// the socket for this link
	socket *socket

	// the lease for this link
	lease *pb.Lease

	// length and weight of the link
	mtx sync.RWMutex

	// determines the cost of the link
	// based on queue length and roundtrip
	length int
	weight int
}

// link methods

// bring up the link
func (l *link) up() error {
	// TODO: manage the length/weight of the link
	return l.socket.accept()
}

// kill the link
func (l *link) down() error {
	return l.socket.close()
}

func (l *link) Length() int {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	return l.length
}

func (l *link) Weight() int {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	return l.weight
}

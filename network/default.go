package network

import (
	"sync"

	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/network/proxy"
	"github.com/micro/go-micro/network/router"
)

type network struct {
	options.Options

	// router
	r router.Router

	// proxy
	p proxy.Proxy

	// id of this network
	id string

	// links maintained for this network
	mtx   sync.RWMutex
	links []Link
}

type node struct {
	*network

	// address of this node
	address string
}

type link struct {
	// the embedded node
	*node

	// length and weight of the link
	mtx    sync.RWMutex
	length int
	weight int
}

// network methods

func (n *network) Id() string {
	return n.id
}

func (n *network) Connect() (Node, error) {
	return nil, nil
}

func (n *network) Peer(Network) (Link, error) {
	return nil, nil
}

func (n *network) Links() ([]Link, error) {
	n.mtx.RLock()
	defer n.mtx.RUnlock()
	return n.links, nil
}

// node methods

func (n *node) Address() string {
	return n.address
}

func (n *node) Close() error {
	return nil
}

func (n *node) Accept() (*Message, error) {
	return nil, nil
}

func (n *node) Send(*Message) error {
	return nil
}

// link methods

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

package socket

import (
	"sync"
)

type Pool struct {
	sync.RWMutex
	pool map[string]*Socket
}

func (p *Pool) Get(id string) (*Socket, bool) {
	// attempt to get existing socket
	p.RLock()
	socket, ok := p.pool[id]
	if ok {
		p.RUnlock()
		return socket, ok
	}
	p.RUnlock()

	// create new socket
	socket = New(id)
	// save socket
	p.Lock()
	p.pool[id] = socket
	p.Unlock()
	// return socket
	return socket, false
}

func (p *Pool) Release(s *Socket) {
	p.Lock()
	defer p.Unlock()

	// close the socket
	s.Close()
	delete(p.pool, s.id)
}

// Close the pool and delete all the sockets
func (p *Pool) Close() {
	p.Lock()
	defer p.Unlock()
	for id, sock := range p.pool {
		sock.Close()
		delete(p.pool, id)
	}
}

// NewPool returns a new socket pool
func NewPool() *Pool {
	return &Pool{
		pool: make(map[string]*Socket),
	}
}

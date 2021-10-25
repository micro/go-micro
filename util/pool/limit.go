package pool

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go-micro.dev/v4/transport"
)

// DefaultMaxIdleConnsPer is the default value of Transport's
// MaxIdleConnsPerHost.
const DefaultMaxIdleConnsPer = 2

type limitPool struct {
	// MaxIdleConns controls the maximum number of idle (keep-alive)
	// connections across all hosts. Zero means no limit.
	MaxIdleConns int

	// MaxIdleConnsPer, if non-zero, controls the maximum idle
	// (keep-alive) connections to keep per-host. If zero,
	// DefaultMaxIdleConnsPer is used.
	MaxIdleConnsPer int

	// MaxConnsPer optionally limits the total number of
	// connections per host, including connections in the dialing,
	// active, and idle states. On limit violation, dials will block.
	//
	// Zero means no limit.
	MaxConnsPer int

	// IdleConnTimeout is the maximum amount of time an idle
	// (keep-alive) connection will remain idle before closing
	// itself.
	// Zero means no limit.
	IdleConnTimeout time.Duration

	idleMu      sync.Mutex
	idleConns   map[string][]*persistConn
	idlePerWait map[string]wantConnQueue
	idleLRU     connLRU

	connsMu      sync.Mutex
	connsPer     map[string]int
	connsPerWait map[string]wantConnQueue

	tr transport.Transport
}

func newLimitPool(opts Options) *limitPool {
	pool := &limitPool{
		tr: opts.Transport,

		MaxIdleConns:    opts.MaxIdleConns,
		MaxIdleConnsPer: opts.MaxIdleConnsPer,
		MaxConnsPer:     opts.MaxConnsPer,

		IdleConnTimeout: opts.IdleConnTimeout,
	}

	if pool.MaxIdleConnsPer == 0 {
		pool.MaxIdleConnsPer = DefaultMaxIdleConnsPer
	}

	return pool
}

// Get
func (p *limitPool) Get(addr string, opts ...transport.DialOption) (_ Conn, err error) {
	options := &transport.DialOptions{}
	for _, opt := range opts {
		opt(options)
	}

	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	cancel := func() {}
	if options.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
	}
	defer cancel()

	w := &wantConn{
		key:  fmt.Sprintf("%s|%s", p.tr.String(), addr),
		addr: addr,
		ctx:  ctx,

		ready: make(chan struct{}, 1),
	}
	defer func() {
		if err != nil {
			w.cancel(p, err)
		}
	}()

	if ok := p.queueForIdleConn(w); ok {
		pc := w.pc
		return pc, nil
	}

	p.queueForDial(w)

	select {
	case <-w.ready:
		if w.err != nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}
		return w.pc, w.err
	case <-ctx.Done():
		w.cancel(p, ctx.Err())
		return nil, w.err
	}
}

// queueForIdleConn
func (p *limitPool) queueForIdleConn(w *wantConn) (delivered bool) {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()

	if pconns, ok := p.idleConns[w.key]; ok {
		var notBefore time.Time
		if p.IdleConnTimeout > 0 {
			notBefore = time.Now().Add(-p.IdleConnTimeout)
		}

		stop := false
		delivered := false
		for len(pconns) > 0 && !stop {
			pc := pconns[len(pconns)-1]
			isTimeout := !notBefore.IsZero() && pc.idleAt.Round(0).Before(notBefore)
			if isTimeout {
				// Async cleanup. Launch in its own goroutine (as if a
				// time.AfterFunc called it); it acquires idleMu, which we're
				// holding, and does a synchronous Conn.Close.
				go pc.closeConnIfStillIdle()
				pconns = pconns[:len(pconns)-1]
				continue
			}
			delivered = w.tryDeliver(pc, nil)
			if delivered {
				p.idleLRU.remove(pc)
				pconns = pconns[:len(pconns)-1]
			}
			stop = true
		}
		if len(pconns) > 0 {
			p.idleConns[w.key] = pconns
		} else {
			delete(p.idleConns, w.key)
		}
		if stop {
			return delivered
		}
	}

	if p.idlePerWait == nil {
		p.idlePerWait = make(map[string]wantConnQueue)
	}
	q := p.idlePerWait[w.key]
	q.cleanFront()
	q.pushBack(w)
	p.idlePerWait[w.key] = q
	return false
}

// put idle conn
func (p *limitPool) putOrCloseIdleConn(pconn *persistConn) {
	if err := p.tryPutIdleConn(pconn); err != nil {
		pconn.Client.Close()
	}
}

func (p *limitPool) tryPutIdleConn(pconn *persistConn) error {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()

	key := pconn.cacheKey
	if q, ok := p.idlePerWait[key]; ok {
		done := false
		for q.len() > 0 {
			w := q.popFront()
			if w.tryDeliver(pconn, nil) {
				done = true
				break
			}
		}
		if q.len() == 0 {
			delete(p.idlePerWait, key)
		} else {
			p.idlePerWait[key] = q
		}
		if done {
			return nil
		}
	}

	if p.idleConns == nil {
		p.idleConns = make(map[string][]*persistConn)
	}

	conns := p.idleConns[key]
	if len(conns) >= p.MaxIdleConnsPer {
		return errors.New("too many idle connections")
	}

	p.idleConns[key] = append(conns, pconn)
	p.idleLRU.add(pconn)
	if p.MaxIdleConns != 0 && p.idleLRU.len() > p.MaxIdleConns {
		oldest := p.idleLRU.removeOldest()
		oldest.Client.Close()
		p.removeIdleConnLocked(oldest)
	}

	// TODO: idleTimer
	pconn.idleAt = time.Now()
	return nil
}

func (p *limitPool) removeIdleConnLocked(pc *persistConn) bool {
	p.idleLRU.remove(pc)
	key := pc.cacheKey
	pconns := p.idleConns[key]

	removed := false
	switch len(pconns) {
	case 0:
		// NOTHING
	case 1:
		if pconns[0] == pc {
			delete(p.idleConns, key)
			removed = true
		}
	default:
		for i, v := range pconns {
			if v != pc {
				continue
			}

			copy(pconns[i:], pconns[i+1:])
			p.idleConns[key] = pconns[:len(pconns)-1]
			removed = true
			break
		}
	}
	return removed
}

// queueForDial
func (p *limitPool) queueForDial(w *wantConn) {
	if p.MaxConnsPer <= 0 {
		go p.dialConnFor(w)
		return
	}

	p.connsMu.Lock()
	defer p.connsMu.Unlock()

	if n := p.connsPer[w.key]; n < p.MaxConnsPer {
		if p.connsPer == nil {
			p.connsPer = make(map[string]int)
		}
		p.connsPer[w.key] = n + 1
		go p.dialConnFor(w)
		return
	}

	if p.connsPerWait == nil {
		p.connsPerWait = make(map[string]wantConnQueue)
	}
	q := p.connsPerWait[w.key]
	q.cleanFront()
	q.pushBack(w)
	p.connsPerWait[w.key] = q
}

func (p *limitPool) dialConnFor(w *wantConn) {
	pc, err := p.dialConn(w.ctx, w.addr, w.opts...)
	delivered := w.tryDeliver(pc, err)
	if err == nil && !delivered {
		p.putOrCloseIdleConn(pc)
	}
	if err != nil {
		p.decConnsPer(w.key)
	}
}

func (p *limitPool) dialConn(ctx context.Context, addr string, opts ...transport.DialOption) (*persistConn, error) {
	client, err := p.tr.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}

	return &persistConn{
		Client: client,

		id:        uuid.New().String(),
		createdAt: time.Now(),

		pool:     p,
		cacheKey: fmt.Sprintf("%s|%s", p.tr.String(), addr),
	}, nil
}

func (p *limitPool) Close() error {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()

	for _, conns := range p.idleConns {
		for _, conn := range conns {
			conn.Client.Close()
		}
	}

	return nil
}

func (p *limitPool) Release(pc Conn, err error) error {
	if err != nil {
		return pc.(*persistConn).Client.Close()
	}
	if err := p.tryPutIdleConn(pc.(*persistConn)); err != nil {
		return pc.(*persistConn).Client.Close()
	}
	return nil
}

// release conn
func (p *limitPool) decConnsPer(key string) {
	if p.MaxConnsPer <= 0 {
		return
	}

	p.connsMu.Lock()
	defer p.connsMu.Unlock()
	n := p.connsPer[key]
	if n == 0 {
		panic("pool: internal error: connCount underflow")
	}

	if q := p.connsPerWait[key]; q.len() > 0 {
		done := false
		for q.len() > 0 {
			w := q.popFront()
			if w.waiting() {
				go p.dialConnFor(w)
				done = true
				break
			}
		}
		if q.len() == 0 {
			delete(p.connsPerWait, key)
		} else {
			p.connsPerWait[key] = q
		}
		if done {
			return
		}
	}

	if n--; n == 0 {
		delete(p.connsPer, key)
	} else {
		p.connsPer[key] = n
	}
}

type wantConn struct {
	key string

	ctx  context.Context
	addr string
	opts []transport.DialOption

	ready chan struct{}
	pc    *persistConn
	err   error

	mu sync.Mutex
}

func (w *wantConn) tryDeliver(pc *persistConn, err error) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.pc != nil || w.err != nil {
		return false
	}

	w.pc, w.err = pc, err
	close(w.ready)
	return true
}

func (w *wantConn) waiting() bool {
	select {
	case <-w.ready:
		return false
	default:
		return true
	}
}

// cancel marks w as no longer wanting a result (for example, due to cancellation).
// If a connection has been delivered already,
// cancel returns it with t.putOrCloseIdleConn.
func (w *wantConn) cancel(pool *limitPool, err error) {
	w.mu.Lock()
	if w.pc == nil && w.err == nil {
		close(w.ready)
	}
	pc := w.pc
	w.pc = nil
	w.err = err
	w.mu.Unlock()

	if pc != nil {
		pool.putOrCloseIdleConn(pc)
	}
}

type persistConn struct {
	transport.Client

	id        string
	createdAt time.Time

	cacheKey string
	pool     *limitPool
	idleAt   time.Time
}

func (pc *persistConn) closeConnIfStillIdle() {
	p := pc.pool
	p.idleMu.Lock()
	defer p.idleMu.Unlock()
	if _, ok := p.idleLRU.m[pc]; !ok {
		return
	}
	p.removeIdleConnLocked(pc)
	_ = pc.Close()
}

// TODO: where us this method?
func (pc *persistConn) Closed() error {
	return pc.Close()
}

func (pc *persistConn) Id() string {
	return pc.id
}

func (pc *persistConn) Created() time.Time {
	return pc.createdAt
}

// A wantConnQueue is a queue of wantConns.
type wantConnQueue struct {
	// This is a queue, not a deque.
	// It is split into two stages - head[headPos:] and tail.
	// popFront is trivial (headPos++) on the first stage, and
	// pushBack is trivial (append) on the second stage.
	// If the first stage is empty, popFront can swap the
	// first and second stages to remedy the situation.
	//
	// This two-stage split is analogous to the use of two lists
	// in Okasaki's purely functional queue but without the
	// overhead of reversing the list when swapping stages.
	head    []*wantConn
	headPos int
	tail    []*wantConn
}

// len returns the number of items in the queue.
func (q *wantConnQueue) len() int {
	return len(q.head) - q.headPos + len(q.tail)
}

// pushBack adds w to the back of the queue.
func (q *wantConnQueue) pushBack(w *wantConn) {
	q.tail = append(q.tail, w)
}

// popFront removes and returns the wantConn at the front of the queue.
func (q *wantConnQueue) popFront() *wantConn {
	if q.headPos >= len(q.head) {
		if len(q.tail) == 0 {
			return nil
		}
		// Pick up tail as new head, clear tail.
		q.head, q.headPos, q.tail = q.tail, 0, q.head[:0]
	}
	w := q.head[q.headPos]
	q.head[q.headPos] = nil
	q.headPos++
	return w
}

// peekFront returns the wantConn at the front of the queue without removing it.
func (q *wantConnQueue) peekFront() *wantConn {
	if q.headPos < len(q.head) {
		return q.head[q.headPos]
	}
	if len(q.tail) > 0 {
		return q.tail[0]
	}
	return nil
}

// cleanFront pops any wantConns that are no longer waiting from the head of the
// queue, reporting whether any were popped.
func (q *wantConnQueue) cleanFront() (cleaned bool) {
	for {
		w := q.peekFront()
		if w == nil || w.waiting() {
			return cleaned
		}
		q.popFront()
		cleaned = true
	}
}

type connLRU struct {
	ll *list.List // list.Element.Value type is of *persistConn
	m  map[*persistConn]*list.Element
}

// add adds pc to the head of the linked list.
func (cl *connLRU) add(pc *persistConn) {
	if cl.ll == nil {
		cl.ll = list.New()
		cl.m = make(map[*persistConn]*list.Element)
	}
	ele := cl.ll.PushFront(pc)
	if _, ok := cl.m[pc]; ok {
		panic("persistConn was already in LRU")
	}
	cl.m[pc] = ele
}

func (cl *connLRU) removeOldest() *persistConn {
	ele := cl.ll.Back()
	pc := ele.Value.(*persistConn)
	cl.ll.Remove(ele)
	delete(cl.m, pc)
	return pc
}

// remove removes pc from cl.
func (cl *connLRU) remove(pc *persistConn) {
	if ele, ok := cl.m[pc]; ok {
		cl.ll.Remove(ele)
		delete(cl.m, pc)
	}
}

// len returns the number of items in the cache.
func (cl *connLRU) len() int {
	return len(cl.m)
}

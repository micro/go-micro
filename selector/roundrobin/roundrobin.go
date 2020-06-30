package roundrobin

import (
	"sort"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/router"
	"github.com/micro/go-micro/v2/selector"
)

var routeTTL = time.Minute * 15

// NewSelector returns an initalised round robin selector
func NewSelector(opts ...selector.Option) selector.Selector {
	r := &roundrobin{
		routes: make(map[uint64]time.Time),
		ticker: time.NewTicker(time.Minute),
	}
	go r.cleanRoutes()
	return r
}

type roundrobin struct {
	ticker *time.Ticker

	// routes is a map with the key being a route's hash and the value being the last time it
	// was used to perform a request
	routes map[uint64]time.Time
	sync.Mutex
}

func (r *roundrobin) Init(opts ...selector.Option) error {
	return nil
}

func (r *roundrobin) Options() selector.Options {
	return selector.Options{}
}

func (r *roundrobin) Select(routes []*router.Route) (*router.Route, error) {
	if len(routes) == 0 {
		return nil, selector.ErrNoneAvailable
	}

	r.Lock()
	defer r.Unlock()

	// setLastUsed will update the last used time for a route
	setLastUsed := func(hash uint64) {
		r.routes[hash] = time.Now()
	}

	// calculate the route hashes once
	hashes := make(map[*router.Route]uint64, len(routes))
	for _, s := range routes {
		hashes[s] = s.Hash()
	}

	// if a route hasn't yet been seen, prioritise it
	for srv, hash := range hashes {
		if _, ok := r.routes[hash]; !ok {
			setLastUsed(hash)
			return srv, nil
		}
	}

	// sort the services by the time they were last used
	sort.SliceStable(routes, func(i, j int) bool {
		iLastSeen := r.routes[hashes[routes[i]]]
		jLastSeen := r.routes[hashes[routes[j]]]
		return iLastSeen.UnixNano() < jLastSeen.UnixNano()
	})

	// return the route which was last used
	setLastUsed(hashes[routes[0]])
	return routes[0], nil
}

func (r *roundrobin) Record(srv *router.Route, err error) error {
	return nil
}

func (r *roundrobin) Close() error {
	r.ticker.Stop()
	return nil
}

func (r *roundrobin) String() string {
	return "roundrobin"
}

func (r *roundrobin) cleanRoutes() {
	for {
		// watch for ticks until the ticker is closed
		if _, ok := <-r.ticker.C; !ok {
			return
		}

		r.Lock()

		// copy the slice to prevent concurrent map iteration and map write
		rts := r.routes

		for hash, t := range rts {
			if t.Unix() < time.Now().Add(-routeTTL).Unix() {
				delete(r.routes, hash)
			}
		}
		r.Unlock()
	}
}

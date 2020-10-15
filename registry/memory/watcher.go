package memory

import (
	"errors"

	"github.com/micro/go-micro/v3/registry"
)

type Watcher struct {
	id   string
	wo   registry.WatchOptions
	res  chan *registry.Result
	exit chan bool
}

func (m *Watcher) Next() (*registry.Result, error) {
	for {
		select {
		case r := <-m.res:
			if r.Service == nil {
				continue
			}

			if len(m.wo.Service) > 0 && m.wo.Service != r.Service.Name {
				continue
			}

			// extract domain from service metadata
			var domain string
			if r.Service.Metadata != nil && len(r.Service.Metadata["domain"]) > 0 {
				domain = r.Service.Metadata["domain"]
			} else {
				domain = registry.DefaultDomain
			}

			// only send the event if watching the wildcard or this specific domain
			if m.wo.Domain == registry.WildcardDomain || m.wo.Domain == domain {
				return r, nil
			}
		case <-m.exit:
			return nil, errors.New("watcher stopped")
		}
	}
}

func (m *Watcher) Stop() {
	select {
	case <-m.exit:
		return
	default:
		close(m.exit)
	}
}

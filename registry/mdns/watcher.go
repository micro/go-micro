package mdns

import (
	"errors"
	"strings"

	"github.com/micro/go-micro/registry"
	"github.com/micro/mdns"
)

type mdnsWatcher struct {
	ch   chan *mdns.ServiceEntry
	exit chan struct{}
}

func (m *mdnsWatcher) Next() (*registry.Result, error) {
	for {
		select {
		case e := <-m.ch:
			txt, err := decode(e.InfoFields)
			if err != nil {
				continue
			}

			if len(txt.Service) == 0 || len(txt.Version) == 0 {
				continue
			}

			var action string

			if e.TTL == 0 {
				action = "delete"
			} else {
				action = "create"
			}

			service := &registry.Service{
				Name:      txt.Service,
				Version:   txt.Version,
				Endpoints: txt.Endpoints,
			}

			// TODO: don't hardcode .local.
			if !strings.HasSuffix(e.Name, "."+service.Name+".local.") {
				continue
			}

			service.Nodes = append(service.Nodes, &registry.Node{
				Id:       strings.TrimSuffix(e.Name, "."+service.Name+".local."),
				Address:  e.AddrV4.String(),
				Port:     e.Port,
				Metadata: txt.Metadata,
			})

			return &registry.Result{
				Action:  action,
				Service: service,
			}, nil
		case <-m.exit:
			return nil, errors.New("watcher stopped")
		}
	}
}

func (m *mdnsWatcher) Stop() {
	select {
	case <-m.exit:
		return
	default:
		close(m.exit)
	}
}

package nats

import (
	"encoding/json"
	"time"

	"github.com/asim/go-micro/v3/registry"
	"github.com/nats-io/nats.go"
)

type natsWatcher struct {
	sub *nats.Subscription
	wo  registry.WatchOptions
}

func (n *natsWatcher) Next() (*registry.Result, error) {
	var result *registry.Result
	for {
		m, err := n.sub.NextMsg(time.Minute)
		if err != nil && err == nats.ErrTimeout {
			continue
		} else if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(m.Data, &result); err != nil {
			return nil, err
		}
		if len(n.wo.Service) > 0 && result.Service.Name != n.wo.Service {
			continue
		}
		break
	}

	return result, nil
}

func (n *natsWatcher) Stop() {
	n.sub.Unsubscribe()
}

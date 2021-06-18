package nacos

import (
	"time"

	"github.com/asim/go-micro/v3/config/encoder"
	"github.com/asim/go-micro/v3/config/source"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type watcher struct {
	configClient  config_client.IConfigClient
	e             encoder.Encoder
	name          string
	group, dataId string

	ch   chan *source.ChangeSet
	exit chan bool
}

func newConfigWatcher(cc config_client.IConfigClient, e encoder.Encoder, name, group, dataId string) (source.Watcher, error) {
	w := &watcher{
		e:            e,
		name:         name,
		configClient: cc,
		group:        group,
		dataId:       dataId,
		ch:           make(chan *source.ChangeSet),
		exit:         make(chan bool),
	}

	err := w.configClient.ListenConfig(vo.ConfigParam{
		DataId:   dataId,
		Group:    group,
		OnChange: w.callback,
	})

	return w, err
}

func (w *watcher) callback(namespace, group, dataId, data string) {
	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Format:    w.e.String(),
		Source:    w.name,
		Data:      []byte(data),
	}
	cs.Checksum = cs.Sum()

	w.ch <- cs
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	select {
	case cs := <-w.ch:
		return cs, nil
	case <-w.exit:
		return nil, source.ErrWatcherStopped
	}
}

func (w *watcher) Stop() error {
	select {
	case <-w.exit:
		return nil
	default:
		close(w.exit)
	}

	return nil
}

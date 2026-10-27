package etcd

import (
	"testing"

	"go-micro.dev/v6/registry"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestWatcherSkipsDeleteWithoutPrevKV(t *testing.T) {
	service := &registry.Service{Name: "example", Version: "1.0.0"}
	watchCh := make(chan clientv3.WatchResponse, 1)
	watchCh <- clientv3.WatchResponse{Events: []*clientv3.Event{
		{
			Type: mvccpb.DELETE,
			Kv:   &mvccpb.KeyValue{Key: []byte("missing-prev-kv")},
		},
		{
			Type:   mvccpb.DELETE,
			Kv:     &mvccpb.KeyValue{Key: []byte("valid-delete")},
			PrevKv: &mvccpb.KeyValue{Value: []byte(encode(service))},
		},
	}}

	watcher := &etcdWatcher{
		stop: make(chan bool, 1),
		w:    watchCh,
	}

	result, err := watcher.Next()
	if err != nil {
		t.Fatalf("Next() returned an unexpected error: %v", err)
	}
	if result.Action != "delete" {
		t.Fatalf("Next() action = %q, want %q", result.Action, "delete")
	}
	if result.Service.Name != service.Name || result.Service.Version != service.Version {
		t.Fatalf("Next() service = %#v, want %#v", result.Service, service)
	}
}

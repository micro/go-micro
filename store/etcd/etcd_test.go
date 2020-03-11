package etcd

import "testing"

func TestEtcd(t *testing.T) {
	e := NewStore()
	if err := e.Init(); err != nil {
		t.Error(err)
	}
}

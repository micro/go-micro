package segmentio_test

import (
	"os"
	"strings"
	"testing"

	"github.com/asim/go-micro/v3/broker"
	segmentio "github.com/asim/go-micro/plugins/broker/segmentio/v3"
)

var (
	bm = &broker.Message{
		Header: map[string]string{"hkey": "hval"},
		Body:   []byte("body"),
	}
)

func TestPubSub(t *testing.T) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		t.Skip()
	}

	var addrs []string
	if addr := os.Getenv("BROKER_ADDRS"); len(addr) == 0 {
		addrs = []string{"127.0.0.1:9092"}
	} else {
		addrs = strings.Split(addr, ",")
	}

	b := segmentio.NewBroker(broker.Addrs(addrs...))
	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}
	defer func() {
		if err := b.Disconnect(); err != nil {
			t.Fatal(err)
		}
	}()

	done := make(chan bool, 1)
	fn := func(msg broker.Event) error {
		done <- true
		return msg.Ack()
	}

	sub, err := b.Subscribe("test_topic", fn, broker.Queue("test"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := sub.Unsubscribe(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := b.Publish("test_topic", bm); err != nil {
		t.Fatal(err)
	}
	<-done
}

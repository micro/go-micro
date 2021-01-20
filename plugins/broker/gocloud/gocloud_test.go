package gocloud

import (
	"fmt"
	"testing"
	"time"

	"github.com/asim/go-micro/v3/broker"
)

func TestBroker(t *testing.T) {
	const nMessages = 10
	b := NewBroker()
	donec := make(chan struct{})
	sub, err := receiveN(b, "T", nMessages, donec)
	if err != nil {
		t.Fatal(err)
	}
	if err := publishN(b, "T", nMessages); err != nil {
		t.Fatal(err)
	}
	<-donec
	sub.Unsubscribe()
}

func publishN(b broker.Broker, topic string, n int) error {
	for i := 0; i < n; i++ {
		msg := &broker.Message{
			Header: map[string]string{
				"id": fmt.Sprintf("%d", i),
			},
			Body: []byte(fmt.Sprintf("%d: %v", i, time.Now())),
		}
		if err := b.Publish(topic, msg); err != nil {
			return err
		}
	}
	return nil
}

func receiveN(b broker.Broker, topic string, n int, donec chan struct{}) (broker.Subscriber, error) {
	r := 0
	handler := func(p broker.Event) error {
		r++
		p.Ack()
		if r >= n {
			close(donec)
		}
		return nil
	}
	return b.Subscribe(topic, handler, broker.Queue("S"))
}

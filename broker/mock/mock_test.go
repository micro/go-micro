package mock

import (
	"fmt"
	"testing"

	"github.com/micro/go-micro/broker"
)

func TestBroker(t *testing.T) {
	b := NewBroker()

	if err := b.Connect(); err != nil {
		t.Fatal("Unexpected connect error %v", err)
	}

	topic := "test"
	count := 10

	fn := func(p broker.Publication) error {
		m := p.Message()
		t.Logf("Received message id %s %+v for topic %s", m.Header["id"], m, p.Topic())
		return nil
	}

	sub, err := b.Subscribe(topic, fn)
	if err != nil {
		t.Fatalf("Unexpected error subscribing %v", err)
	}

	for i := 0; i < count; i++ {
		message := &broker.Message{
			Header: map[string]string{
				"foo": "bar",
				"id":  fmt.Sprintf("%d", i),
			},
			Body: []byte(`hello world`),
		}

		t.Logf("Sending message %d %+v for topic %s", i, message, topic)
		if err := b.Publish(topic, message); err != nil {
			t.Fatalf("Unexpected error publishing %d", i)
		}
	}

	if err := sub.Unsubscribe(); err != nil {
		t.Fatalf("Unexpected error unsubscribing from %s: %v", topic, err)
	}

	if err := b.Disconnect(); err != nil {
		t.Fatalf("Unexpected connect error %v", err)
	}
}

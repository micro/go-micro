package broker

import (
	"testing"

	"github.com/micro/go-micro/registry/mock"
)

func TestBroker(t *testing.T) {
	m := mock.NewRegistry()
	b := NewBroker([]string{}, Registry(m))

	if err := b.Init(); err != nil {
		t.Errorf("Unexpected init error: %v", err)
	}

	if err := b.Connect(); err != nil {
		t.Errorf("Unexpected connect error: %v", err)
	}

	msg := &Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	done := make(chan bool)

	sub, err := b.Subscribe("test", func(p Publication) error {
		m := p.Message()
		t.Logf("Received message %+v", m)

		if string(m.Body) != string(msg.Body) {
			t.Errorf("Unexpected msg %s, expected %s", string(m.Body), string(msg.Body))
		}

		close(done)
		return nil
	})
	if err != nil {
		t.Errorf("Unexpected subscribe error: %v", err)
	}

	if err := b.Publish("test", msg); err != nil {
		t.Errorf("Unexpected publish error: %v", err)
	}

	<-done
	sub.Unsubscribe()

	if err := b.Disconnect(); err != nil {
		t.Errorf("Unexpected disconnect error: %v", err)
	}
}

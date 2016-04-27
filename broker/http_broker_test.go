package broker

import (
	"sync"
	"testing"

	"github.com/micro/go-micro/registry/mock"
)

func TestBroker(t *testing.T) {
	m := mock.NewRegistry()
	b := NewBroker(Registry(m))

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

func TestConcurrentSubBroker(t *testing.T) {
	m := mock.NewRegistry()
	b := NewBroker(Registry(m))

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

	var subs []Subscriber
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		sub, err := b.Subscribe("test", func(p Publication) error {
			defer wg.Done()

			m := p.Message()

			if string(m.Body) != string(msg.Body) {
				t.Errorf("Unexpected msg %s, expected %s", string(m.Body), string(msg.Body))
			}

			return nil
		})
		if err != nil {
			t.Errorf("Unexpected subscribe error: %v", err)
		}

		wg.Add(1)
		subs = append(subs, sub)
	}

	if err := b.Publish("test", msg); err != nil {
		t.Errorf("Unexpected publish error: %v", err)
	}

	wg.Wait()

	for _, sub := range subs {
		sub.Unsubscribe()
	}

	if err := b.Disconnect(); err != nil {
		t.Errorf("Unexpected disconnect error: %v", err)
	}
}

func TestConcurrentPubBroker(t *testing.T) {
	m := mock.NewRegistry()
	b := NewBroker(Registry(m))

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

	var wg sync.WaitGroup

	sub, err := b.Subscribe("test", func(p Publication) error {
		defer wg.Done()

		m := p.Message()

		if string(m.Body) != string(msg.Body) {
			t.Errorf("Unexpected msg %s, expected %s", string(m.Body), string(msg.Body))
		}

		return nil
	})
	if err != nil {
		t.Errorf("Unexpected subscribe error: %v", err)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)

		if err := b.Publish("test", msg); err != nil {
			t.Errorf("Unexpected publish error: %v", err)
		}
	}

	wg.Wait()

	sub.Unsubscribe()

	if err := b.Disconnect(); err != nil {
		t.Errorf("Unexpected disconnect error: %v", err)
	}
}

package redis

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/asim/go-micro/v3/broker"
)

func subscribe(t *testing.T, b broker.Broker, topic string, handle broker.Handler) broker.Subscriber {
	s, err := b.Subscribe(topic, handle)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func publish(t *testing.T, b broker.Broker, topic string, msg *broker.Message) {
	if err := b.Publish(topic, msg); err != nil {
		t.Fatal(err)
	}
}

func unsubscribe(t *testing.T, s broker.Subscriber) {
	if err := s.Unsubscribe(); err != nil {
		t.Fatal(err)
	}
}

func TestBroker(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not defined")
	}

	b := NewBroker(broker.Addrs(url))

	// Only setting options.
	b.Init()

	if err := b.Connect(); err != nil {
		t.Fatal(err)
	}
	defer b.Disconnect()

	// Large enough buffer to not block.
	msgs := make(chan string, 10)

	go func() {
		s1 := subscribe(t, b, "test", func(p broker.Event) error {
			m := p.Message()
			msgs <- fmt.Sprintf("s1:%s", string(m.Body))
			return nil
		})

		s2 := subscribe(t, b, "test", func(p broker.Event) error {
			m := p.Message()
			msgs <- fmt.Sprintf("s2:%s", string(m.Body))
			return nil
		})

		publish(t, b, "test", &broker.Message{
			Body: []byte("hello"),
		})

		publish(t, b, "test", &broker.Message{
			Body: []byte("world"),
		})

		unsubscribe(t, s1)

		publish(t, b, "test", &broker.Message{
			Body: []byte("other"),
		})

		unsubscribe(t, s2)

		publish(t, b, "test", &broker.Message{
			Body: []byte("none"),
		})

		close(msgs)
	}()

	var actual []string
	for msg := range msgs {
		actual = append(actual, msg)
	}

	exp := []string{
		"s1:hello",
		"s2:hello",
		"s1:world",
		"s2:world",
		"s2:other",
	}

	// Order is not guaranteed.
	sort.Strings(actual)
	sort.Strings(exp)

	if !reflect.DeepEqual(actual, exp) {
		t.Fatalf("expected %v, got %v", exp, actual)
	}
}

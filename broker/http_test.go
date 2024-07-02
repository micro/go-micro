package broker_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/registry"
)

var (
	// mock data.
	testData = map[string][]*registry.Service{
		"foo": {
			{
				Name:    "foo",
				Version: "1.0.0",
				Nodes: []*registry.Node{
					{
						Id:      "foo-1.0.0-123",
						Address: "localhost:9999",
					},
					{
						Id:      "foo-1.0.0-321",
						Address: "localhost:9999",
					},
				},
			},
			{
				Name:    "foo",
				Version: "1.0.1",
				Nodes: []*registry.Node{
					{
						Id:      "foo-1.0.1-321",
						Address: "localhost:6666",
					},
				},
			},
			{
				Name:    "foo",
				Version: "1.0.3",
				Nodes: []*registry.Node{
					{
						Id:      "foo-1.0.3-345",
						Address: "localhost:8888",
					},
				},
			},
		},
	}
)

func newTestRegistry() registry.Registry {
	return registry.NewMemoryRegistry(registry.Services(testData))
}

func sub(b *testing.B, c int) {
	b.StopTimer()
	m := newTestRegistry()

	brker := broker.NewBroker(broker.Registry(m))
	topic := uuid.New().String()

	if err := brker.Init(); err != nil {
		b.Fatalf("Unexpected init error: %v", err)
	}

	if err := brker.Connect(); err != nil {
		b.Fatalf("Unexpected connect error: %v", err)
	}

	msg := &broker.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	var subs []broker.Subscriber
	done := make(chan bool, c)

	for i := 0; i < c; i++ {
		sub, err := brker.Subscribe(topic, func(p broker.Event) error {
			done <- true
			m := p.Message()

			if string(m.Body) != string(msg.Body) {
				b.Fatalf("Unexpected msg %s, expected %s", string(m.Body), string(msg.Body))
			}

			return nil
		}, broker.Queue("shared"))
		if err != nil {
			b.Fatalf("Unexpected subscribe error: %v", err)
		}
		subs = append(subs, sub)
	}

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		if err := brker.Publish(topic, msg); err != nil {
			b.Fatalf("Unexpected publish error: %v", err)
		}
		<-done
		b.StopTimer()
	}

	for _, sub := range subs {
		if err := sub.Unsubscribe(); err != nil {
			b.Fatalf("Unexpected unsubscribe error: %v", err)
		}
	}

	if err := brker.Disconnect(); err != nil {
		b.Fatalf("Unexpected disconnect error: %v", err)
	}
}

func pub(b *testing.B, c int) {
	b.StopTimer()
	m := newTestRegistry()
	brk := broker.NewBroker(broker.Registry(m))
	topic := uuid.New().String()

	if err := brk.Init(); err != nil {
		b.Fatalf("Unexpected init error: %v", err)
	}

	if err := brk.Connect(); err != nil {
		b.Fatalf("Unexpected connect error: %v", err)
	}

	msg := &broker.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	done := make(chan bool, c*4)

	sub, err := brk.Subscribe(topic, func(p broker.Event) error {
		done <- true
		m := p.Message()
		if string(m.Body) != string(msg.Body) {
			b.Fatalf("Unexpected msg %s, expected %s", string(m.Body), string(msg.Body))
		}
		return nil
	}, broker.Queue("shared"))
	if err != nil {
		b.Fatalf("Unexpected subscribe error: %v", err)
	}

	var wg sync.WaitGroup
	ch := make(chan int, c*4)
	b.StartTimer()

	for i := 0; i < c; i++ {
		go func() {
			for range ch {
				if err := brk.Publish(topic, msg); err != nil {
					b.Fatalf("Unexpected publish error: %v", err)
				}
				select {
				case <-done:
				case <-time.After(time.Second):
				}
				wg.Done()
			}
		}()
	}

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		ch <- i
	}

	wg.Wait()
	b.StopTimer()
	sub.Unsubscribe()
	close(ch)
	close(done)

	if err := brk.Disconnect(); err != nil {
		b.Fatalf("Unexpected disconnect error: %v", err)
	}
}

func TestBroker(t *testing.T) {
	m := newTestRegistry()
	b := broker.NewBroker(broker.Registry(m))

	if err := b.Init(); err != nil {
		t.Fatalf("Unexpected init error: %v", err)
	}

	if err := b.Connect(); err != nil {
		t.Fatalf("Unexpected connect error: %v", err)
	}

	msg := &broker.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	done := make(chan bool)

	sub, err := b.Subscribe("test", func(p broker.Event) error {
		m := p.Message()

		if string(m.Body) != string(msg.Body) {
			t.Fatalf("Unexpected msg %s, expected %s", string(m.Body), string(msg.Body))
		}

		close(done)
		return nil
	})
	if err != nil {
		t.Fatalf("Unexpected subscribe error: %v", err)
	}

	if err := b.Publish("test", msg); err != nil {
		t.Fatalf("Unexpected publish error: %v", err)
	}

	<-done
	if err := sub.Unsubscribe(); err != nil {
		t.Fatalf("Unexpected unsubscribe error: %v", err)
	}

	if err := b.Disconnect(); err != nil {
		t.Fatalf("Unexpected disconnect error: %v", err)
	}
}

func TestConcurrentSubBroker(t *testing.T) {
	m := newTestRegistry()
	b := broker.NewBroker(broker.Registry(m))

	if err := b.Init(); err != nil {
		t.Fatalf("Unexpected init error: %v", err)
	}

	if err := b.Connect(); err != nil {
		t.Fatalf("Unexpected connect error: %v", err)
	}

	msg := &broker.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	var subs []broker.Subscriber
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		sub, err := b.Subscribe("test", func(p broker.Event) error {
			defer wg.Done()

			m := p.Message()

			if string(m.Body) != string(msg.Body) {
				t.Fatalf("Unexpected msg %s, expected %s", string(m.Body), string(msg.Body))
			}

			return nil
		})
		if err != nil {
			t.Fatalf("Unexpected subscribe error: %v", err)
		}

		wg.Add(1)
		subs = append(subs, sub)
	}

	if err := b.Publish("test", msg); err != nil {
		t.Fatalf("Unexpected publish error: %v", err)
	}

	wg.Wait()

	for _, sub := range subs {
		if err := sub.Unsubscribe(); err != nil {
			t.Fatalf("Unexpected unsubscribe error: %v", err)
		}
	}

	if err := b.Disconnect(); err != nil {
		t.Fatalf("Unexpected disconnect error: %v", err)
	}
}

func TestConcurrentPubBroker(t *testing.T) {
	m := newTestRegistry()
	b := broker.NewBroker(broker.Registry(m))

	if err := b.Init(); err != nil {
		t.Fatalf("Unexpected init error: %v", err)
	}

	if err := b.Connect(); err != nil {
		t.Fatalf("Unexpected connect error: %v", err)
	}

	msg := &broker.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	var wg sync.WaitGroup

	sub, err := b.Subscribe("test", func(p broker.Event) error {
		defer wg.Done()

		m := p.Message()

		if string(m.Body) != string(msg.Body) {
			t.Fatalf("Unexpected msg %s, expected %s", string(m.Body), string(msg.Body))
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Unexpected subscribe error: %v", err)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)

		if err := b.Publish("test", msg); err != nil {
			t.Fatalf("Unexpected publish error: %v", err)
		}
	}

	wg.Wait()

	if err := sub.Unsubscribe(); err != nil {
		t.Fatalf("Unexpected unsubscribe error: %v", err)
	}

	if err := b.Disconnect(); err != nil {
		t.Fatalf("Unexpected disconnect error: %v", err)
	}
}

func BenchmarkSub1(b *testing.B) {
	sub(b, 1)
}
func BenchmarkSub8(b *testing.B) {
	sub(b, 8)
}

func BenchmarkSub32(b *testing.B) {
	sub(b, 32)
}

func BenchmarkSub64(b *testing.B) {
	sub(b, 64)
}

func BenchmarkSub128(b *testing.B) {
	sub(b, 128)
}

func BenchmarkPub1(b *testing.B) {
	pub(b, 1)
}

func BenchmarkPub8(b *testing.B) {
	pub(b, 8)
}

func BenchmarkPub32(b *testing.B) {
	pub(b, 32)
}

func BenchmarkPub64(b *testing.B) {
	pub(b, 64)
}

func BenchmarkPub128(b *testing.B) {
	pub(b, 128)
}

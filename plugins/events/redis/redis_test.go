package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"go-micro.dev/v4/events"
	"github.com/stretchr/testify/assert"
)

type testObj struct {
	One string
	Two int64
}

func TestStream(t *testing.T) {
	opts := []Option{
		Address("localhost:6379"),
		User("default"),
	}

	start := time.Now()
	s, err := NewStream(opts...)
	assert.NoError(t, err, "Should be no error when creating stream")
	t.Run("Simple", func(t *testing.T) {
		ch, err := s.Consume("foo")
		assert.NoError(t, err, "Should be no error subscribing")
		testPubCons := func(topic string, tobj testObj) {
			err = s.Publish(topic, tobj, events.WithMetadata(map[string]string{"meta": "bar"}))

			assert.NoError(t, err, "Unexpected error publishing")
			var ev events.Event
			select {
			case <-time.After(5 * time.Second):
				t.Errorf("Failed to receive message within the time limit")
			case ev = <-ch:

			}
			assert.NotEmpty(t, ev.ID, "Missing ID")
			assert.NotEmpty(t, ev.Timestamp, "Missing Timestamp")
			assert.Equal(t, topic, ev.Topic, "Incorrect topic")
			assert.Len(t, ev.Metadata, 1)
			assert.Equal(t, "bar", ev.Metadata["meta"])

			var tes testObj
			assert.NoError(t, json.Unmarshal(ev.Payload, &tes))
			assert.Equal(t, tobj.One, tes.One)
			assert.Equal(t, tobj.Two, tes.Two)
		}

		testPubCons("foo", testObj{
			One: "foo",
			Two: 12345,
		})

		testPubCons("foo", testObj{
			One: "bar",
			Two: 6789,
		})
	})

	t.Run("Offset", func(t *testing.T) {
		// test offset
		ch, err := s.Consume("foo", events.WithOffset(start))
		assert.NoError(t, err)
		// should have 2 messages
		for i := 0; i < 2; i++ {
			select {
			case <-time.After(5 * time.Second):
				t.Errorf("Failed to receive message within the time limit")
			case <-ch:

			}
		}
		select {
		case <-time.After(5 * time.Second):
		case <-ch:
			t.Errorf("Failed to receive message within the time limit")
		}

	})
	// test retry limit
	t.Run("RetryLimit", func(t *testing.T) {

		ch, err := s.Consume("fooretry", events.WithOffset(start), events.WithRetryLimit(3))
		assert.NoError(t, err)

		err = s.Publish("fooretry", testObj{
			One: "1",
			Two: 2,
		}, events.WithMetadata(map[string]string{"meta": "bar"}))

		assert.NoError(t, err, "Unexpected error publishing")

		id := ""
		ts := time.Time{}
	loop:
		for i := 0; i <= 3; i++ {
			var ev events.Event
			select {
			case <-time.After(5 * time.Second):
				t.Errorf("Failed to receive message within the time limit, loop %d", i)
				break loop
			case ev = <-ch:
				ev.Nack()
			}
			if len(id) == 0 {
				id = ev.ID
				ts = ev.Timestamp
			} else {
				assert.Equal(t, id, ev.ID)
				assert.Equal(t, ts.Unix(), ev.Timestamp.Unix())
			}
			assert.NotEmpty(t, ev.Timestamp, "Missing Timestamp")
			assert.Equal(t, "fooretry", ev.Topic, "Incorrect topic")
			assert.Len(t, ev.Metadata, 1)
			assert.Equal(t, "bar", ev.Metadata["meta"])
		}

		select {
		case <-time.After(5 * time.Second):
		case ev := <-ch:
			t.Errorf("Received unexpected message %+v", ev)
		}
	})

	t.Run("WithGroup", func(t *testing.T) {
		topic := "foogroup"
		assert.NoError(t, s.(*redisStream).redisClient.XTrim(context.Background(), "stream-"+topic, 0).Err())
		ch1, err := s.Consume(topic, events.WithGroup("mygroup"))
		assert.NoError(t, err)

		ch2, err := s.Consume(topic, events.WithGroup("mygroup"))
		assert.NoError(t, err)

		seen := map[string]bool{}
		for i := 0; i < 100; i++ {
			err = s.Publish(topic, testObj{
				One: fmt.Sprintf("%d", i),
				Two: int64(i),
			})
			assert.NoError(t, err)
			seen[fmt.Sprintf("%d", i)] = true
		}
		ch1Processed := false
		ch2Processed := false

	loop:
		for {
			var tobj testObj
			// ch1 should have first message and ch2 should have second message
			select {
			case ev1 := <-ch1:
				assert.NoError(t, json.Unmarshal(ev1.Payload, &tobj))
				ch1Processed = true
				if !seen[tobj.One] {
					t.Errorf("Already processed this event %+v", tobj)
					break loop
				}
				ev1.Ack()
			case ev2 := <-ch2:
				assert.NoError(t, json.Unmarshal(ev2.Payload, &tobj))
				ch2Processed = true
				if !seen[tobj.One] {
					t.Errorf("Already processed this event %+v", tobj)
					break loop
				}
				ev2.Ack()
			case <-time.After(5 * time.Second):
				if len(seen) == 0 {
					break loop
				}
				t.Errorf("Timed out waiting for event")
				break loop
			}
			delete(seen, tobj.One)

		}
		assert.True(t, ch1Processed)
		assert.True(t, ch2Processed)
	})

}

func TestCleanup(t *testing.T) {
	opts := []Option{
		Address("localhost:6379"),
		User("default"),
	}

	// make the timeouts quick so we're not waiting ages for the test
	consumerTimeout = 2 * time.Second
	readGroupTimeout = 2 * time.Second
	s, err := NewStream(opts...)

	assert.NoError(t, err, "Should be no error when creating stream")
	topic := "fooclean"
	assert.NoError(t, s.(*redisStream).redisClient.XTrim(context.Background(), "stream-"+topic, 0).Err())
	_, err = s.Consume(topic, events.WithGroup("mygroup"))
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	assert.NoError(t, s.Publish(topic, testObj{}))
	cons, err := s.(*redisStream).redisClient.XInfoConsumers(context.Background(), "stream-"+topic, "mygroup").Result()
	assert.NoError(t, err)
	assert.Len(t, cons, 1)
	time.Sleep(5 * time.Second)
	cons, err = s.(*redisStream).redisClient.XInfoConsumers(context.Background(), "stream-"+topic, "mygroup").Result()
	assert.NoError(t, err)
	assert.Len(t, cons, 0)

}

func TestJanitor(t *testing.T) {
	opts := []Option{
		Address("localhost:6379"),
		User("default"),
	}

	// make the timeouts quick so we're not waiting ages for the test
	readGroupTimeout = 1 * time.Second
	janitorConsumerTimeout = 8 * time.Second
	janitorFrequency = 1 * time.Second
	consumerTimeout = 1 * time.Second
	s, err := NewStream(opts...)
	assert.NoError(t, err, "Should be no error when creating stream")
	assert.NoError(t, s.(*redisStream).redisClient.FlushDB(context.Background()).Err())
	topic := "foojanitor"
	_, err = s.Consume(topic, events.WithGroup("mygroup"))
	assert.NoError(t, err)
	ch, err := s.Consume(topic, events.WithGroup("mygroup"))
	assert.NoError(t, err)
	go func() {
		for {
			ev := <-ch
			t.Logf("Received on 2")
			if len(ev.ID) == 0 {
				continue
			}
			ev.Ack()

		}
	}()
	for i := 0; i < 10; i++ {
		assert.NoError(t, s.Publish(topic, testObj{}))
	}

	go func() {
		for i := 0; i < 10; i++ {
			assert.NoError(t, s.Publish(topic, testObj{}))
			time.Sleep(1 * time.Second)
		}
	}()
	cons, err := s.(*redisStream).redisClient.XInfoConsumers(context.Background(), "stream-"+topic, "mygroup").Result()
	assert.NoError(t, err)
	assert.Len(t, cons, 2)
	time.Sleep(10 * time.Second)
	cons, err = s.(*redisStream).redisClient.XInfoConsumers(context.Background(), "stream-"+topic, "mygroup").Result()
	assert.NoError(t, err)
	assert.Len(t, cons, 1)

}

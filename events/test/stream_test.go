// +build nats

package test

import (
	"sync"
	"testing"
	"time"

	"github.com/asim/go-micro/v3/events"
	"github.com/asim/go-micro/v3/events/memory"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type testPayload struct {
	Message string
}

type testCase struct {
	str  events.Stream
	name string
}

func TestStream(t *testing.T) {
	tcs := []testCase{}

	stream, err := memory.NewStream()
	assert.Nilf(t, err, "NewStream should not return an error")
	assert.NotNilf(t, stream, "NewStream should return a stream object")
	tcs = append(tcs, testCase{str: stream, name: "memory"})

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTestStream(t, tc.str)
		})
	}

}

func runTestStream(t *testing.T, stream events.Stream) {
	// TestMissingTopic will test the topic validation on publish
	t.Run("TestMissingTopic", func(t *testing.T) {
		err := stream.Publish("", nil)
		assert.Equalf(t, err, events.ErrMissingTopic, "Publishing to a blank topic should return an error")
	})

	// TestConsumeTopic will publish a message to the test topic. The subscriber will subscribe to the
	// same test topic.
	t.Run("TestConsumeTopic", func(t *testing.T) {
		payload := &testPayload{Message: "HelloWorld"}
		metadata := map[string]string{"foo": "bar"}

		// create the subscriber
		evChan, err := stream.Consume("test")
		assert.Nilf(t, err, "Consume should not return an error")

		// setup the subscriber async
		var wg sync.WaitGroup

		go func() {
			timeout := time.NewTimer(time.Millisecond * 250)

			select {
			case event, _ := <-evChan:
				assert.NotNilf(t, event, "The message was nil")
				assert.Equal(t, event.Metadata, metadata, "Metadata didn't match")

				var result testPayload
				err = event.Unmarshal(&result)
				assert.Nil(t, err, "Error decoding result")
				assert.Equal(t, result, *payload, "Payload didn't match")

				wg.Done()
			case <-timeout.C:
				t.Fatalf("Event was not recieved")
			}
		}()

		err = stream.Publish("test", payload, events.WithMetadata(metadata))
		assert.Nil(t, err, "Publishing a valid message should not return an error")
		wg.Add(1)

		// wait for the subscriber to recieve the message or timeout
		wg.Wait()
	})

	// TestConsumeQueue will publish a message to a random topic. Two subscribers will then consume
	// the message from the firehose topic with different queues. The second subscriber will be registered
	// after the message is published to test durability.
	t.Run("TestConsumeQueue", func(t *testing.T) {
		topic := uuid.New().String()
		payload := &testPayload{Message: "HelloWorld"}
		metadata := map[string]string{"foo": "bar"}

		// create the first subscriber
		evChan1, err := stream.Consume(topic)
		assert.Nilf(t, err, "Consume should not return an error")

		// setup the subscriber async
		var wg sync.WaitGroup

		go func() {
			timeout := time.NewTimer(time.Millisecond * 250)

			select {
			case event, _ := <-evChan1:
				assert.NotNilf(t, event, "The message was nil")
				assert.Equal(t, event.Metadata, metadata, "Metadata didn't match")

				var result testPayload
				err = event.Unmarshal(&result)
				assert.Nil(t, err, "Error decoding result")
				assert.Equal(t, result, *payload, "Payload didn't match")

				wg.Done()
			case <-timeout.C:
				t.Fatalf("Event was not recieved")
			}
		}()

		err = stream.Publish(topic, payload, events.WithMetadata(metadata))
		assert.Nil(t, err, "Publishing a valid message should not return an error")
		wg.Add(2)

		// create the second subscriber
		evChan2, err := stream.Consume(topic,
			events.WithQueue("second_queue"),
			events.WithStartAtTime(time.Now().Add(time.Minute*-1)),
		)
		assert.Nilf(t, err, "Consume should not return an error")

		go func() {
			timeout := time.NewTimer(time.Second * 1)

			select {
			case event, _ := <-evChan2:
				assert.NotNilf(t, event, "The message was nil")
				assert.Equal(t, event.Metadata, metadata, "Metadata didn't match")

				var result testPayload
				err = event.Unmarshal(&result)
				assert.Nil(t, err, "Error decoding result")
				assert.Equal(t, result, *payload, "Payload didn't match")

				wg.Done()
			case <-timeout.C:
				t.Fatalf("Event was not recieved")
			}
		}()

		// wait for the subscriber to recieve the message or timeout
		wg.Wait()
	})

	t.Run("AckingNacking", func(t *testing.T) {
		ch, err := stream.Consume("foobarAck", events.WithAutoAck(false, 5*time.Second))
		assert.NoError(t, err, "Unexpected error subscribing")
		assert.NoError(t, stream.Publish("foobarAck", map[string]string{"foo": "message 1"}))
		assert.NoError(t, stream.Publish("foobarAck", map[string]string{"foo": "message 2"}))

		ev := <-ch
		ev.Ack()
		ev = <-ch
		nacked := ev.ID
		ev.Nack()
		select {
		case ev = <-ch:
			assert.Equal(t, ev.ID, nacked, "Nacked message should have been received again")
			assert.NoError(t, ev.Ack())
		case <-time.After(7 * time.Second):
			t.Fatalf("Timed out waiting for message to be put back on queue")
		}

	})

	t.Run("Retries", func(t *testing.T) {
		ch, err := stream.Consume("foobarRetries", events.WithAutoAck(false, 5*time.Second), events.WithRetryLimit(1))
		assert.NoError(t, err, "Unexpected error subscribing")
		assert.NoError(t, stream.Publish("foobarRetries", map[string]string{"foo": "message 1"}))

		ev := <-ch
		id := ev.ID
		ev.Nack()
		ev = <-ch
		assert.Equal(t, id, ev.ID, "Nacked message should have been received again")
		ev.Nack()
		select {
		case ev = <-ch:
			t.Fatalf("Unexpected event received")
		case <-time.After(7 * time.Second):
		}

	})

	t.Run("InfiniteRetries", func(t *testing.T) {
		ch, err := stream.Consume("foobarRetriesInf", events.WithAutoAck(false, 2*time.Second))
		assert.NoError(t, err, "Unexpected error subscribing")
		assert.NoError(t, stream.Publish("foobarRetriesInf", map[string]string{"foo": "message 1"}))

		count := 0
		id := ""
		for {
			select {
			case ev := <-ch:
				if id != "" {
					assert.Equal(t, id, ev.ID, "Nacked message should have been received again")
				}
				id = ev.ID
			case <-time.After(3 * time.Second):
				t.Fatalf("Unexpected event received")
			}

			count++
			if count == 11 {
				break
			}
		}

	})

	t.Run("twoSubs", func(t *testing.T) {
		ch1, err := stream.Consume("foobarTwoSubs1", events.WithAutoAck(false, 5*time.Second))
		assert.NoError(t, err, "Unexpected error subscribing to topic 1")
		ch2, err := stream.Consume("foobarTwoSubs2", events.WithAutoAck(false, 5*time.Second))
		assert.NoError(t, err, "Unexpected error subscribing to topic 2")

		assert.NoError(t, stream.Publish("foobarTwoSubs2", map[string]string{"foo": "message 1"}))
		assert.NoError(t, stream.Publish("foobarTwoSubs1", map[string]string{"foo": "message 1"}))

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			ev := <-ch1
			assert.Equal(t, "foobarTwoSubs1", ev.Topic, "Received message from unexpected topic")
			wg.Done()
		}()
		go func() {
			ev := <-ch2
			assert.Equal(t, "foobarTwoSubs2", ev.Topic, "Received message from unexpected topic")
			wg.Done()
		}()
		wg.Wait()
	})
}

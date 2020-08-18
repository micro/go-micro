package memory

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v3/events"
	"github.com/stretchr/testify/assert"
)

type testPayload struct {
	Message string
}

func TestStream(t *testing.T) {
	stream, err := NewStream()
	assert.Nilf(t, err, "NewStream should not return an error")
	assert.NotNilf(t, stream, "NewStream should return a stream object")

	// TestMissingTopic will test the topic validation on publish
	t.Run("TestMissingTopic", func(t *testing.T) {
		err := stream.Publish("")
		assert.Equalf(t, err, events.ErrMissingTopic, "Publishing to a blank topic should return an error")
	})

	// TestFirehose will publish a message to the test topic. The subscriber will subscribe to the
	// firehose topic (indicated by a lack of the topic option).
	t.Run("TestFirehose", func(t *testing.T) {
		payload := &testPayload{Message: "HelloWorld"}
		metadata := map[string]string{"foo": "bar"}

		// create the subscriber
		evChan, err := stream.Subscribe()
		assert.Nilf(t, err, "Subscribe should not return an error")

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

		err = stream.Publish("test",
			events.WithPayload(payload),
			events.WithMetadata(metadata),
		)
		assert.Nil(t, err, "Publishing a valid message should not return an error")
		wg.Add(1)

		// wait for the subscriber to recieve the message or timeout
		wg.Wait()
	})

	// TestSubscribeTopic will publish a message to the test topic. The subscriber will subscribe to the
	// same test topic.
	t.Run("TestSubscribeTopic", func(t *testing.T) {
		payload := &testPayload{Message: "HelloWorld"}
		metadata := map[string]string{"foo": "bar"}

		// create the subscriber
		evChan, err := stream.Subscribe(events.WithTopic("test"))
		assert.Nilf(t, err, "Subscribe should not return an error")

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

		err = stream.Publish("test",
			events.WithPayload(payload),
			events.WithMetadata(metadata),
		)
		assert.Nil(t, err, "Publishing a valid message should not return an error")
		wg.Add(1)

		// wait for the subscriber to recieve the message or timeout
		wg.Wait()
	})

	// TestSubscribeQueue will publish a message to a random topic. Two subscribers will then consume
	// the message from the firehose topic with different queues. The second subscriber will be registered
	// after the message is published to test durability.
	t.Run("TestSubscribeQueue", func(t *testing.T) {
		topic := uuid.New().String()
		payload := &testPayload{Message: "HelloWorld"}
		metadata := map[string]string{"foo": "bar"}

		// create the first subscriber
		evChan1, err := stream.Subscribe(events.WithTopic(topic))
		assert.Nilf(t, err, "Subscribe should not return an error")

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

		err = stream.Publish(topic,
			events.WithPayload(payload),
			events.WithMetadata(metadata),
		)
		assert.Nil(t, err, "Publishing a valid message should not return an error")
		wg.Add(2)

		// create the second subscriber
		evChan2, err := stream.Subscribe(
			events.WithTopic(topic),
			events.WithQueue("second_queue"),
			events.WithStartAtTime(time.Now().Add(time.Minute*-1)),
		)
		assert.Nilf(t, err, "Subscribe should not return an error")

		go func() {
			timeout := time.NewTimer(time.Millisecond * 250)

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
}

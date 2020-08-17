package nats

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	stan "github.com/nats-io/stan.go"
	"github.com/pkg/errors"

	"github.com/micro/go-micro/v3/events"
	"github.com/micro/go-micro/v3/logger"
)

const (
	defaultClusterID = "micro"
	eventsTopic      = "events"
)

// NewStream returns an initialized nats stream or an error if the connection to the nats
// server could not be established
func NewStream(opts ...Option) (events.Stream, error) {
	// parse the options
	options := Options{
		ClientID:  uuid.New().String(),
		ClusterID: defaultClusterID,
	}
	for _, o := range opts {
		o(&options)
	}

	// pass the address as an option if it was set
	var cOpts []stan.Option
	if len(options.Address) > 0 {
		cOpts = append(cOpts, stan.NatsURL(options.Address))
	}

	// connect to the cluster
	conn, err := stan.Connect(options.ClusterID, options.ClientID, cOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error connecting to nats")
	}

	return &stream{conn}, nil
}

type stream struct {
	conn stan.Conn
}

// Publish a message to a topic
func (s *stream) Publish(topic string, opts ...events.PublishOption) error {
	// validate the topic
	if len(topic) == 0 {
		return events.ErrMissingTopic
	}

	// parse the options
	options := events.PublishOptions{
		Timestamp: time.Now(),
	}
	for _, o := range opts {
		o(&options)
	}

	// encode the message if it's not already encoded
	var payload []byte
	if p, ok := options.Payload.([]byte); ok {
		payload = p
	} else {
		p, err := json.Marshal(options.Payload)
		if err != nil {
			return events.ErrEncodingMessage
		}
		payload = p
	}

	// construct the event
	event := &events.Event{
		ID:        uuid.New().String(),
		Topic:     topic,
		Timestamp: options.Timestamp,
		Metadata:  options.Metadata,
		Payload:   payload,
	}

	// serialize the event to bytes
	bytes, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "Error encoding event")
	}

	// publish the event to the events channel
	if _, err := s.conn.PublishAsync(eventsTopic, bytes, nil); err != nil {
		return errors.Wrap(err, "Error publishing message to events")
	}

	// publish the event to the topic's channel
	if _, err := s.conn.PublishAsync(event.Topic, bytes, nil); err != nil {
		return errors.Wrap(err, "Error publishing message to topic")
	}

	return nil
}

// Subscribe to a topic
func (s *stream) Subscribe(opts ...events.SubscribeOption) (<-chan events.Event, error) {
	// parse the options
	options := events.SubscribeOptions{
		Topic: eventsTopic,
		Queue: uuid.New().String(),
	}
	for _, o := range opts {
		o(&options)
	}

	// setup the subscriber
	c := make(chan events.Event)
	handleMsg := func(m *stan.Msg) {
		// decode the message
		var evt events.Event
		if err := json.Unmarshal(m.Data, &evt); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Errorf("Error decoding message: %v", err)
			}
			// not ackknowledging the message is the way to indicate an error occured
			return
		}

		// push onto the channel and wait for the consumer to take the event off before we acknowledge it.
		c <- evt

		if err := m.Ack(); err != nil && logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("Error acknowledging message: %v", err)
		}
	}

	// setup the options
	subOpts := []stan.SubscriptionOption{
		stan.DurableName(options.Topic),
		stan.SetManualAckMode(),
	}
	if options.StartAtTime.Unix() > 0 {
		stan.StartAtTime(options.StartAtTime)
	}

	// connect the subscriber
	_, err := s.conn.QueueSubscribe(options.Topic, options.Queue, handleMsg, subOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error subscribing to topic")
	}

	return c, nil
}

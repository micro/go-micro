// Package natsjs provides a NATS Jetstream implementation of the events.Stream interface.
package natsjs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	nats "github.com/nats-io/nats.go"
	"github.com/pkg/errors"

	"go-micro.dev/v5/events"
	"go-micro.dev/v5/logger"
)

const (
	defaultClusterID = "micro"
)

// NewStream returns an initialized nats stream or an error if the connection to the nats
// server could not be established.
func NewStream(opts ...Option) (events.Stream, error) {
	// parse the options
	options := Options{
		ClientID:  uuid.New().String(),
		ClusterID: defaultClusterID,
		Logger:    logger.DefaultLogger,
	}
	for _, o := range opts {
		o(&options)
	}

	s := &stream{opts: options}

	natsJetStreamCtx, err := connectToNatsJetStream(options)
	if err != nil {
		return nil, fmt.Errorf("error connecting to nats cluster %v: %w", options.ClusterID, err)
	}

	s.natsJetStreamCtx = natsJetStreamCtx

	return s, nil
}

type stream struct {
	opts             Options
	natsJetStreamCtx nats.JetStreamContext
}

func connectToNatsJetStream(options Options) (nats.JetStreamContext, error) {
	nopts := nats.GetDefaultOptions()
	if options.TLSConfig != nil {
		nopts.Secure = true
		nopts.TLSConfig = options.TLSConfig
	}

	if options.NkeyConfig != "" {
		nopts.Nkey = options.NkeyConfig
	}

	if len(options.Address) > 0 {
		nopts.Servers = strings.Split(options.Address, ",")
	}

	if options.Name != "" {
		nopts.Name = options.Name
	}

	if options.Username != "" && options.Password != "" {
		nopts.User = options.Username
		nopts.Password = options.Password
	}

	conn, err := nopts.Connect()
	if err != nil {
		tls := nopts.TLSConfig != nil
		return nil, fmt.Errorf("error connecting to nats at %v with tls enabled (%v): %w", options.Address, tls, err)
	}

	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("error while obtaining JetStream context: %w", err)
	}

	return js, nil
}

// Publish a message to a topic.
func (s *stream) Publish(topic string, msg interface{}, opts ...events.PublishOption) error {
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
	if p, ok := msg.([]byte); ok {
		payload = p
	} else {
		p, err := json.Marshal(msg)
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

	// publish the event to the topic's channel
	// publish synchronously if configured
	if s.opts.SyncPublish {
		_, err := s.natsJetStreamCtx.Publish(event.Topic, bytes)
		if err != nil {
			err = errors.Wrap(err, "Error publishing message to topic")
		}

		return err
	}

	// publish asynchronously by default
	if _, err := s.natsJetStreamCtx.PublishAsync(event.Topic, bytes); err != nil {
		return errors.Wrap(err, "Error publishing message to topic")
	}

	return nil
}

// Consume from a topic.
func (s *stream) Consume(topic string, opts ...events.ConsumeOption) (<-chan events.Event, error) {
	// validate the topic
	if len(topic) == 0 {
		return nil, events.ErrMissingTopic
	}

	log := s.opts.Logger

	// parse the options
	options := events.ConsumeOptions{
		Group: uuid.New().String(),
	}
	for _, o := range opts {
		o(&options)
	}

	// setup the subscriber
	channel := make(chan events.Event)
	handleMsg := func(msg *nats.Msg) {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		// decode the message
		var evt events.Event
		if err := json.Unmarshal(msg.Data, &evt); err != nil {
			log.Logf(logger.ErrorLevel, "Error decoding message: %v", err)
			// not acknowledging the message is the way to indicate an error occurred
			return
		}

		if !options.AutoAck {
			// set up the ack funcs
			evt.SetAckFunc(func() error {
				return msg.Ack()
			})
			evt.SetNackFunc(func() error {
				return msg.Nak()
			})
		}

		// push onto the channel and wait for the consumer to take the event off before we acknowledge it.
		channel <- evt

		if !options.AutoAck {
			return
		}

		if err := msg.Ack(nats.Context(ctx)); err != nil {
			log.Logf(logger.ErrorLevel, "Error acknowledging message: %v", err)
		}
	}

	// ensure that a stream exists for that topic
	_, err := s.natsJetStreamCtx.StreamInfo(topic)
	if err != nil {
		cfg := &nats.StreamConfig{
			Name: topic,
		}

		_, err = s.natsJetStreamCtx.AddStream(cfg)
		if err != nil {
			return nil, errors.Wrap(err, "Stream did not exist and adding a stream failed")
		}
	}

	// setup the options
	subOpts := []nats.SubOpt{}

	if options.CustomRetries {
		subOpts = append(subOpts, nats.MaxDeliver(options.GetRetryLimit()))
	}

	if options.AutoAck {
		subOpts = append(subOpts, nats.AckNone())
	} else {
		subOpts = append(subOpts, nats.AckExplicit())
	}

	if !options.Offset.IsZero() {
		subOpts = append(subOpts, nats.StartTime(options.Offset))
	} else {
		subOpts = append(subOpts, nats.DeliverNew())
	}

	if options.AckWait > 0 {
		subOpts = append(subOpts, nats.AckWait(options.AckWait))
	}

	// connect the subscriber via a queue group only if durable streams are enabled
	if !s.opts.DisableDurableStreams {
		subOpts = append(subOpts, nats.Durable(options.Group))
		_, err = s.natsJetStreamCtx.QueueSubscribe(topic, options.Group, handleMsg, subOpts...)
	} else {
		subOpts = append(subOpts, nats.ConsumerName(options.Group))
		_, err = s.natsJetStreamCtx.Subscribe(topic, handleMsg, subOpts...)
	}

	if err != nil {
		return nil, errors.Wrap(err, "Error subscribing to topic")
	}

	return channel, nil
}

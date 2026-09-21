// Package natsjs provides a NATS Jetstream implementation of the events.Stream interface.
package natsjs

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	nats "github.com/nats-io/nats.go"
	"github.com/pkg/errors"

	"go-micro.dev/v6/events"
	"go-micro.dev/v6/logger"
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

	conn, natsJetStreamCtx, err := connectToNatsJetStream(options)
	if err != nil {
		return nil, fmt.Errorf("error connecting to nats cluster %v: %w", options.ClusterID, err)
	}

	s.conn = conn
	s.natsJetStreamCtx = natsJetStreamCtx

	return s, nil
}

type stream struct {
	opts             Options
	conn             *nats.Conn // store connection for lifecycle management
	natsJetStreamCtx nats.JetStreamContext
}

func connectToNatsJetStream(options Options) (*nats.Conn, nats.JetStreamContext, error) {
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
		return nil, nil, fmt.Errorf("error connecting to nats at %v with tls enabled (%v): %w", options.Address, tls, err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close() // Close connection if JetStream context fails
		return nil, nil, fmt.Errorf("error while obtaining JetStream context: %w", err)
	}

	return conn, js, nil
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

	if options.ID == "" {
		options.ID = uuid.New().String()
	}

	// construct the event
	event := &events.Event{
		ID:        options.ID,
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
		_, err := s.natsJetStreamCtx.Publish(event.Topic, bytes, nats.MsgId(event.ID))
		if err != nil {
			err = errors.Wrap(err, "Error publishing message to topic")
		}

		return err
	}

	// publish asynchronously by default
	if _, err := s.natsJetStreamCtx.PublishAsync(event.Topic, bytes, nats.MsgId(event.ID)); err != nil {
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
	options := events.ConsumeOptions{AutoAck: true}
	for _, o := range opts {
		o(&options)
	}
	if !s.opts.DisableDurableStreams && options.Group == "" {
		return nil, fmt.Errorf("consumer group is required when durable streams are enabled")
	}
	if s.opts.DisableDurableStreams && options.Group == "" {
		options.Group = uuid.New().String()
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
		// Manual acknowledgements must reach JetStream regardless of AutoAck.
		evt.SetAckFunc(func() error { return msg.Ack() })
		evt.SetNackFunc(func() error { return msg.Nak() })

		// push onto the channel and wait for the consumer to take the event off before we acknowledge it.
		channel <- evt

		if !options.AutoAck {
			return
		}

		if err := msg.Ack(nats.Context(ctx)); err != nil {

			log.Logf(logger.ErrorLevel, "Error acknowledging message: %v", err)
		}
	}

	streamName, err := s.ensureStream(topic)
	if err != nil {
		return nil, err
	}

	// setup the options
	// Disable the NATS callback auto-ack: this callback only delivers to a
	// channel and cannot know when the application has finished processing.
	subOpts := []nats.SubOpt{nats.ManualAck(), nats.BindStream(streamName)}

	if options.CustomRetries {
		subOpts = append(subOpts, nats.MaxDeliver(options.GetRetryLimit()))
	}

	consumerExists := false
	if !s.opts.DisableDurableStreams {
		_, err = s.natsJetStreamCtx.ConsumerInfo(streamName, options.Group)
		switch {
		case err == nil:
			consumerExists = true
		case errors.Is(err, nats.ErrConsumerNotFound):
			// The delivery policy below is used only when creating the durable.
		default:
			return nil, errors.Wrap(err, "Error checking durable consumer")
		}
	}

	// Delivery policies are immutable, so do not specify one when binding to an
	// existing durable. This also keeps consumers created by older versions with
	// DeliverNew compatible after upgrading.
	if !consumerExists {
		// Ack each event independently. Existing durables retain their policy.
		subOpts = append(subOpts, nats.AckExplicit())
		if !options.Offset.IsZero() {
			subOpts = append(subOpts, nats.StartTime(options.Offset))
		} else if !s.opts.DisableDurableStreams {
			subOpts = append(subOpts, nats.DeliverAll())
		} else {
			subOpts = append(subOpts, nats.DeliverNew())
		}
	}

	if options.AckWait > 0 {
		subOpts = append(subOpts, nats.AckWait(options.AckWait))
	}

	// connect the subscriber via a queue group only if durable streams are enabled
	if !s.opts.DisableDurableStreams {
		if consumerExists {
			subOpts = append(subOpts, nats.Bind(streamName, options.Group))
		} else {
			subOpts = append(subOpts, nats.Durable(options.Group))
		}
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

// Close implements io.Closer and closes the underlying NATS connection.
// This method is optional but recommended to prevent connection leaks.
func (s *stream) Close() error {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	return nil
}

// Ensure stream implements io.Closer
var _ io.Closer = (*stream)(nil)

// StreamName maps a topic to a legal stream name while preserving legacy names
// for simple subjects. Dotted subjects use a stable SHA-256-derived name.
func StreamName(topic string) string {
	if !strings.ContainsAny(topic, ".*> /\\\t\r\n") {
		return topic
	}
	return fmt.Sprintf("micro_%x", sha256.Sum256([]byte(topic)))
}

func (s *stream) ensureStream(topic string) (string, error) {
	cfg := nats.StreamConfig{Name: StreamName(topic), Subjects: []string{topic},
		Retention: nats.RetentionPolicy(s.opts.RetentionPolicy), MaxAge: s.opts.MaxAge}
	if s.opts.MaxMsgSize > 0 {
		cfg.MaxMsgSize = int32(s.opts.MaxMsgSize)
	}
	if s.opts.StreamConfig != nil {
		custom, err := s.opts.StreamConfig(topic)
		if err != nil {
			return "", fmt.Errorf("stream config for %s: %w", topic, err)
		}
		cfg = custom
		if cfg.Name == "" {
			cfg.Name = StreamName(topic)
		}
		if len(cfg.Subjects) == 0 {
			cfg.Subjects = []string{topic}
		}
	}
	_, err := s.natsJetStreamCtx.StreamInfo(cfg.Name)
	if err == nil {
		return cfg.Name, nil
	}
	if !errors.Is(err, nats.ErrStreamNotFound) {
		return "", fmt.Errorf("inspect stream %s: %w", cfg.Name, err)
	}
	if _, err = s.natsJetStreamCtx.AddStream(&cfg); err != nil {
		// Another publisher or consumer may have created the same stream meanwhile.
		if errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
			if _, lookupErr := s.natsJetStreamCtx.StreamInfo(cfg.Name); lookupErr == nil {
				return cfg.Name, nil
			}
		}
		return "", fmt.Errorf("create stream %s: %w", cfg.Name, err)
	}
	return cfg.Name, nil
}

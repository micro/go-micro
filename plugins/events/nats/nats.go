package nats

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	"github.com/pkg/errors"

	"go-micro.dev/v4/events"
	"go-micro.dev/v4/logger"
)

const (
	defaultClusterID      = "micro"
	reconnectLoopDuration = 10 * time.Second
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

	s := &stream{opts: options}
	clusterConn, err := connectToStan(options, s.connectionLost)
	if err != nil {
		return nil, fmt.Errorf("error connecting to nats cluster %v: %v", options.ClusterID, err)
	}
	s.conn = clusterConn
	s.status = statusConnected
	return s, nil
}

type streamStatus string

const (
	statusConnected    streamStatus = "CONNECTED"
	statusReconnecting streamStatus = "RECONNECTING"
)

type stream struct {
	opts    Options
	conn    stan.Conn
	connMux sync.Mutex
	status  streamStatus
}

func connectToStan(options Options, handler stan.ConnectionLostHandler) (stan.Conn, error) {
	nopts := nats.GetDefaultOptions()
	if options.TLSConfig != nil {
		nopts.Secure = true
		nopts.TLSConfig = options.TLSConfig
	}
	if len(options.Address) > 0 {
		nopts.Servers = []string{options.Address}
	}
	conn, err := nopts.Connect()
	if err != nil {
		return nil, fmt.Errorf("error connecting to nats at %v with tls enabled (%v): %v", options.Address, nopts.TLSConfig != nil, err)
	}
	// connect to the cluster
	return stan.Connect(options.ClusterID, options.ClientID, stan.NatsConn(conn), stan.SetConnectionLostHandler(handler))
}

func (s *stream) connectionLost(conn stan.Conn, err error) {
	// kick off a new connection
	go func() {
		s.connMux.Lock()
		defer s.connMux.Unlock()
		if s.status == statusReconnecting {
			return
		}
		s.status = statusReconnecting
		// loop until we connect successfully
		for {
			clusterConn, err := connectToStan(s.opts, s.connectionLost)
			if err != nil {
				logger.Errorf("Error trying to reconnect %s. Will retry in %s", err, reconnectLoopDuration)
				time.Sleep(reconnectLoopDuration)
				continue
			}
			logger.Infof("Successfully reconnected to stan cluster")
			s.conn = clusterConn
			s.status = statusConnected
			return
		}

	}()
}

// Publish a message to a topic
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
	if _, err := s.conn.PublishAsync(event.Topic, bytes, nil); err != nil {
		return errors.Wrap(err, "Error publishing message to topic")
	}

	return nil
}

// Consume to a topic
func (s *stream) Consume(topic string, opts ...events.ConsumeOption) (<-chan events.Event, error) {
	// validate the topic
	if len(topic) == 0 {
		return nil, events.ErrMissingTopic
	}

	// parse the options
	options := events.ConsumeOptions{
		Group:   uuid.New().String(),
		AutoAck: true,
	}
	for _, o := range opts {
		o(&options)
	}

	// setup the subscriber
	c := make(chan events.Event)
	handleMsg := func(m *stan.Msg) {
		// poison message handling
		if options.GetRetryLimit() > -1 && m.Redelivered && int(m.RedeliveryCount) > options.GetRetryLimit() {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Errorf("Message retry limit reached, discarding: %v", m.Sequence)
			}
			m.Ack() // ignoring error
			return
		}

		// decode the message
		var evt events.Event
		if err := json.Unmarshal(m.Data, &evt); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Errorf("Error decoding message: %v", err)
			}
			// not acknowledging the message is the way to indicate an error occurred
			return
		}

		if !options.AutoAck {
			// set up the ack funcs
			evt.SetAckFunc(func() error {
				return m.Ack()
			})
			evt.SetNackFunc(func() error {
				// noop. not acknowledging the message is the way to indicate an error occurred
				// we have to wait for the ack wait to kick in before the message is resent
				return nil
			})
		}

		// push onto the channel and wait for the consumer to take the event off before we acknowledge it.
		c <- evt

		if !options.AutoAck {
			return
		}
		if err := m.Ack(); err != nil && logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("Error acknowledging message: %v", err)
		}
	}

	// setup the options
	subOpts := []stan.SubscriptionOption{
		stan.DurableName(topic),
		stan.SetManualAckMode(),
	}
	if options.Offset.Unix() > 0 {
		subOpts = append(subOpts, stan.StartAtTime(options.Offset))
	}
	if options.AckWait > 0 {
		subOpts = append(subOpts, stan.AckWait(options.AckWait))
	}

	// connect the subscriber
	_, err := s.conn.QueueSubscribe(topic, options.Group, handleMsg, subOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error subscribing to topic")
	}

	return c, nil
}

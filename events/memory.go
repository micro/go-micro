package events

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "go-micro.dev/v4/logger"
	"go-micro.dev/v4/store"
)

// NewStream returns an initialized memory stream.
func NewStream(opts ...Option) (Stream, error) {
	// parse the options
	options := NewOptions(opts...)

	return &mem{store: store.NewMemoryStore(), options: options}, nil
}

type subscriber struct {
	Channel chan Event

	retryMap   map[string]int
	Group      string
	Topic      string
	retryLimit int
	ackWait    time.Duration

	sync.RWMutex
	autoAck bool
}

type mem struct {
	options *Options
	store   store.Store

	subs []*subscriber
	sync.RWMutex
}

func (m *mem) Publish(topic string, msg interface{}, opts ...PublishOption) error {
	// validate the topic
	if len(topic) == 0 {
		return ErrMissingTopic
	}

	// parse the options
	options := PublishOptions{
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
			return ErrEncodingMessage
		}
		payload = p
	}

	// construct the event
	event := &Event{
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

	// write to the store
	key := fmt.Sprintf("%v/%v", event.Topic, event.ID)
	if err := m.store.Write(&store.Record{Key: key, Value: bytes}); err != nil {
		return errors.Wrap(err, "Error writing event to store")
	}

	// send to the subscribers async
	go m.handleEvent(event)

	return nil
}

func (m *mem) Consume(topic string, opts ...ConsumeOption) (<-chan Event, error) {
	// validate the topic
	if len(topic) == 0 {
		return nil, ErrMissingTopic
	}

	// parse the options
	options := ConsumeOptions{
		Group:   uuid.New().String(),
		AutoAck: true,
	}
	for _, o := range opts {
		o(&options)
	}
	// TODO RetryLimit

	// setup the subscriber
	sub := &subscriber{
		Channel:    make(chan Event),
		Topic:      topic,
		Group:      options.Group,
		retryMap:   map[string]int{},
		autoAck:    true,
		retryLimit: options.GetRetryLimit(),
	}

	if !options.AutoAck {
		if options.AckWait == 0 {
			return nil, fmt.Errorf("invalid AckWait passed, should be positive integer")
		}
		sub.autoAck = options.AutoAck
		sub.ackWait = options.AckWait
	}

	// register the subscriber
	m.Lock()
	m.subs = append(m.subs, sub)
	m.Unlock()

	// lookup previous events if the start time option was passed
	if options.Offset.Unix() > 0 {
		go m.lookupPreviousEvents(sub, options.Offset)
	}

	// return the channel
	return sub.Channel, nil
}

// lookupPreviousEvents finds events for a subscriber which occurred before a given time and sends
// them into the subscribers channel.
func (m *mem) lookupPreviousEvents(sub *subscriber, startTime time.Time) {
	// lookup all events which match the topic (a blank topic will return all results)
	recs, err := m.store.Read(sub.Topic+"/", store.ReadPrefix())
	if err != nil {
		m.options.Logger.Logf(log.ErrorLevel, "Error looking up previous events: %v", err)
		return
	}

	// loop through the records and send it to the channel if it matches
	for _, r := range recs {
		var ev Event
		if err := json.Unmarshal(r.Value, &ev); err != nil {
			continue
		}
		if ev.Timestamp.Unix() < startTime.Unix() {
			continue
		}
		sendEvent(&ev, sub)
	}
}

// handleEvents sends the event to any registered subscribers.
func (m *mem) handleEvent(ev *Event) {
	m.RLock()
	subs := m.subs
	m.RUnlock()
	logger := m.options.Logger
	// filteredSubs is a KV map of the queue name and subscribers. This is used to prevent a message
	// being sent to two subscribers with the same queue.
	filteredSubs := map[string]*subscriber{}

	// filter down to subscribers who are interested in this topic
	for _, sub := range subs {
		if len(sub.Topic) == 0 || sub.Topic == ev.Topic {
			filteredSubs[sub.Group] = sub
		}
	}

	// send the message to each channel async (since one channel might be blocked)
	for _, sub := range filteredSubs {
		go func(s *subscriber) {
			if err := sendEvent(ev, s); err != nil {
				logger.Log(log.ErrorLevel, err)
			}
		}(sub)
	}
}

func sendEvent(ev *Event, s *subscriber) error {
	evCopy := *ev
	if s.autoAck {
		s.Channel <- evCopy
		return nil
	}
	evCopy.SetAckFunc(ackFunc(s, evCopy))
	evCopy.SetNackFunc(nackFunc(s, evCopy))
	s.retryMap[evCopy.ID] = 0
	tick := time.NewTicker(s.ackWait)
	defer tick.Stop()
	for range tick.C {
		s.Lock()
		count, ok := s.retryMap[evCopy.ID]
		s.Unlock()
		if !ok {
			// success
			break
		}

		if s.retryLimit > -1 && count > s.retryLimit {
			s.Lock()
			delete(s.retryMap, evCopy.ID)
			s.Unlock()
			return fmt.Errorf("Message retry limit reached, discarding: %v %d %d", evCopy.ID, count, s.retryLimit)
		}
		s.Channel <- evCopy
		s.Lock()
		s.retryMap[evCopy.ID] = count + 1
		s.Unlock()
	}
	return nil
}

func ackFunc(s *subscriber, evCopy Event) func() error {
	return func() error {
		s.Lock()
		delete(s.retryMap, evCopy.ID)
		s.Unlock()
		return nil
	}
}

func nackFunc(s *subscriber, evCopy Event) func() error {
	return func() error {
		return nil
	}
}

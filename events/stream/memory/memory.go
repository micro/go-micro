package memory

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v3/events"
	"github.com/micro/go-micro/v3/logger"
	"github.com/micro/go-micro/v3/store"
	"github.com/micro/go-micro/v3/store/memory"
	"github.com/pkg/errors"
)

// NewStream returns an initialized memory stream
func NewStream(opts ...Option) (events.Stream, error) {
	// parse the options
	var options Options
	for _, o := range opts {
		o(&options)
	}
	if options.Store == nil {
		options.Store = memory.NewStore()
	}

	return &mem{store: options.Store}, nil
}

type subscriber struct {
	Queue   string
	Topic   string
	Channel chan events.Event

	sync.RWMutex
	retryMap     map[string]int
	retryLimit   int
	trackRetries bool
	autoAck      bool
	ackWait      time.Duration
}

type mem struct {
	store store.Store

	subs []*subscriber
	sync.RWMutex
}

func (m *mem) Publish(topic string, msg interface{}, opts ...events.PublishOption) error {
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

	// write to the store
	key := fmt.Sprintf("%v/%v", event.Topic, event.ID)
	if err := m.store.Write(&store.Record{Key: key, Value: bytes}); err != nil {
		return errors.Wrap(err, "Error writing event to store")
	}

	// send to the subscribers async
	go m.handleEvent(event)

	return nil
}

func (m *mem) Subscribe(topic string, opts ...events.SubscribeOption) (<-chan events.Event, error) {
	// validate the topic
	if len(topic) == 0 {
		return nil, events.ErrMissingTopic
	}

	// parse the options
	options := events.SubscribeOptions{
		Queue:   uuid.New().String(),
		AutoAck: true,
	}
	for _, o := range opts {
		o(&options)
	}
	// TODO RetryLimit

	// setup the subscriber
	sub := &subscriber{
		Channel:  make(chan events.Event),
		Topic:    topic,
		Queue:    options.Queue,
		retryMap: map[string]int{},
		autoAck:  true,
	}

	if options.CustomRetries {
		sub.trackRetries = true
		sub.retryLimit = options.GetRetryLimit()

	}
	if !options.AutoAck {
		sub.autoAck = options.AutoAck
		sub.ackWait = options.AckWait
	}

	// register the subscriber
	m.Lock()
	m.subs = append(m.subs, sub)
	m.Unlock()

	// lookup previous events if the start time option was passed
	if options.StartAtTime.Unix() > 0 {
		go m.lookupPreviousEvents(sub, options.StartAtTime)
	}

	// return the channel
	return sub.Channel, nil
}

// lookupPreviousEvents finds events for a subscriber which occurred before a given time and sends
// them into the subscribers channel
func (m *mem) lookupPreviousEvents(sub *subscriber, startTime time.Time) {
	// lookup all events which match the topic (a blank topic will return all results)
	recs, err := m.store.Read(sub.Topic+"/", store.ReadPrefix())
	if err != nil && logger.V(logger.ErrorLevel, logger.DefaultLogger) {
		logger.Errorf("Error looking up previous events: %v", err)
		return
	} else if err != nil {
		return
	}

	// loop through the records and send it to the channel if it matches
	for _, r := range recs {
		var ev events.Event
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
func (m *mem) handleEvent(ev *events.Event) {
	m.RLock()
	subs := m.subs
	m.RUnlock()

	// filteredSubs is a KV map of the queue name and subscribers. This is used to prevent a message
	// being sent to two subscribers with the same queue.
	filteredSubs := map[string]*subscriber{}

	// filter down to subscribers who are interested in this topic
	for _, sub := range subs {
		if len(sub.Topic) == 0 || sub.Topic == ev.Topic {
			filteredSubs[sub.Queue] = sub
		}
	}

	// send the message to each channel async (since one channel might be blocked)
	for _, sub := range subs {
		sendEvent(ev, sub)
	}
}

func sendEvent(ev *events.Event, sub *subscriber) {
	go func(s *subscriber) {
		evCopy := *ev
		if s.autoAck {
			s.Channel <- evCopy
			return
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

			if s.trackRetries && count > s.retryLimit {
				if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
					logger.Errorf("Message retry limit reached, discarding: %v %d %d", evCopy.ID, count, s.retryLimit)
				}
				s.Lock()
				delete(s.retryMap, evCopy.ID)
				s.Unlock()
				return
			}
			s.Channel <- evCopy
			s.Lock()
			s.retryMap[evCopy.ID] = count + 1
			s.Unlock()
		}
	}(sub)
}

func ackFunc(s *subscriber, evCopy events.Event) func() error {
	return func() error {
		s.Lock()
		delete(s.retryMap, evCopy.ID)
		s.Unlock()
		return nil
	}
}

func nackFunc(s *subscriber, evCopy events.Event) func() error {
	return func() error {
		return nil
	}
}

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
}

type mem struct {
	store store.Store

	subs []*subscriber
	sync.RWMutex
}

func (m *mem) Publish(topic string, opts ...events.PublishOption) error {
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

	// write to the store
	key := fmt.Sprintf("%v/%v", event.Topic, event.ID)
	if err := m.store.Write(&store.Record{Key: key, Value: bytes}); err != nil {
		return errors.Wrap(err, "Error writing event to store")
	}

	// send to the subscribers async
	go m.handleEvent(event)

	return nil
}

func (m *mem) Subscribe(opts ...events.SubscribeOption) (<-chan events.Event, error) {
	// parse the options
	options := events.SubscribeOptions{
		Queue: uuid.New().String(),
	}
	for _, o := range opts {
		o(&options)
	}

	// setup the subscriber
	sub := &subscriber{
		Channel: make(chan events.Event),
		Topic:   options.Topic,
		Queue:   options.Queue,
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

// lookupPreviousEvents finds events for a subscriber which occured before a given time and sends
// them into the subscribers channel
func (m *mem) lookupPreviousEvents(sub *subscriber, startTime time.Time) {
	var prefix string
	if len(sub.Topic) > 0 {
		prefix = sub.Topic + "/"
	}

	// lookup all events which match the topic (a blank topic will return all results)
	recs, err := m.store.Read(prefix, store.ReadPrefix())
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

		sub.Channel <- ev
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
		go func(s *subscriber) {
			s.Channel <- *ev
		}(sub)
	}
}

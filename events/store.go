package events

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	log "go-micro.dev/v4/logger"
	"go-micro.dev/v4/store"
)

const joinKey = "/"

// NewStore returns an initialized events store.
func NewStore(opts ...StoreOption) Store {
	// parse the options
	var options StoreOptions
	for _, o := range opts {
		o(&options)
	}
	if options.TTL.Seconds() == 0 {
		options.TTL = time.Hour * 24
	}

	options.Logger = log.LoggerOrDefault(options.Logger)

	// return the store
	evs := &evStore{
		opts:  options,
		store: store.NewMemoryStore(),
	}
	if options.Backup != nil {
		go evs.backupLoop()
	}
	return evs
}

type evStore struct {
	opts  StoreOptions
	store store.Store
}

// Read events for a topic.
func (s *evStore) Read(topic string, opts ...ReadOption) ([]*Event, error) {
	// validate the topic
	if len(topic) == 0 {
		return nil, ErrMissingTopic
	}

	// parse the options
	options := ReadOptions{
		Offset: 0,
		Limit:  250,
	}
	for _, o := range opts {
		o(&options)
	}

	// execute the request
	recs, err := s.store.Read(topic+joinKey,
		store.ReadPrefix(),
		store.ReadLimit(options.Limit),
		store.ReadOffset(options.Offset),
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading from store")
	}

	// unmarshal the result
	result := make([]*Event, len(recs))
	for i, r := range recs {
		var e Event
		if err := json.Unmarshal(r.Value, &e); err != nil {
			return nil, errors.Wrap(err, "Invalid event returned from stroe")
		}
		result[i] = &e
	}

	return result, nil
}

// Write an event to the store.
func (s *evStore) Write(event *Event, opts ...WriteOption) error {
	// parse the options
	options := WriteOptions{
		TTL: s.opts.TTL,
	}
	for _, o := range opts {
		o(&options)
	}

	// construct the store record
	bytes, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "Error mashaling event to JSON")
	}
	// suffix event ID with hour resolution for easy retrieval in batches
	timeSuffix := time.Now().Format("2006010215")

	record := &store.Record{
		// key is such that reading by prefix indexes by topic and reading by suffix indexes by time
		Key:    event.Topic + joinKey + event.ID + joinKey + timeSuffix,
		Value:  bytes,
		Expiry: options.TTL,
	}

	// write the record to the store
	if err := s.store.Write(record); err != nil {
		return errors.Wrap(err, "Error writing to the store")
	}

	return nil
}

func (s *evStore) backupLoop() {
	for {
		err := s.opts.Backup.Snapshot(s.store)
		if err != nil {
			s.opts.Logger.Logf(log.ErrorLevel, "Error running backup %s", err)
		}

		time.Sleep(1 * time.Hour)
	}
}

// Backup is an interface for snapshotting the events store to long term storage.
type Backup interface {
	Snapshot(st store.Store) error
}

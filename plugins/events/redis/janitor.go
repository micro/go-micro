package stream

import (
	"context"
	"os"
	"strconv"
	"time"

	"go-micro.dev/v4/logger"
)

var (
	janitorConsumerTimeout = 24 * time.Hour     // threshold for an "old" consumer
	janitorFrequency       = 4 * time.Hour      // how often do we run the janitor
	defaultTrimDuration    = 5 * 24 * time.Hour // oldest event in stream
)

func (r *redisStream) runJanitor() {
	// Some times it's possible that a consumer group has old consumers that have failed to be deleted.
	// Janitor will clean up any consumers that haven't been seen for X duration
	go func() {
		for {
			if err := r.cleanupConsumers(); err != nil {
				logger.Errorf("Error cleaning up consumers")
			}
			time.Sleep(janitorFrequency)
		}
	}()

}

func (r *redisStream) cleanupConsumers() error {
	ctx := context.Background()
	now := time.Now()
	keys, err := r.redisClient.Keys(ctx, "stream-*").Result()
	if err != nil {
		return err
	}
	for _, streamName := range keys {
		logger.Infof("Cleaning up stream %s", streamName)
		s, err := r.redisClient.XInfoStreamFull(ctx, streamName, 1).Result()
		if err != nil {
			logger.Errorf("Error getting info on groups for %s: %s", streamName, err)
			continue
		}
		for _, g := range s.Groups {
			logger.Infof("Cleaning up stream %s group %s", streamName, g.Name)
			for _, c := range g.Consumers {
				// Seen time is the last time this consumer read a message successfully.
				// This means if the stream is low volume you could delete currently connected consumers
				// This isn't a massive problem because the clients should reconnect with a new consumer
				if c.SeenTime.Add(janitorConsumerTimeout).After(now) {
					continue
				}
				logger.Infof("Cleaning up consumer %s, it is %s old", c.Name, time.Since(c.SeenTime))
				if err := r.redisClient.XGroupDelConsumer(ctx, streamName, g.Name, c.Name).Err(); err != nil {
					logger.Errorf("Error deleting consumer %s %s %s: %s", streamName, g.Name, c.Name, err)
					continue
				}
			}

		}
		d := defaultTrimDuration
		durationStr := os.Getenv("MICRO_REDIS_TRIM_DURATION")
		if len(durationStr) > 0 {
			parsed, err := time.ParseDuration(durationStr)
			if err != nil {
				logger.Warnf("Failed to parse MICRO_REDIS_TRIM_DURATION %s", err)
			} else {
				d = parsed
			}
		}

		// `XTRIM MINID` requires Redis 6.2
		minID := strconv.FormatInt(time.Now().Add(-d).Unix()*1000, 10)
		if err := r.redisClient.XTrimMinID(ctx, streamName, minID).Err(); err != nil {
			logger.Errorf("Error trimming %s", err)
		}
	}
	return nil
}

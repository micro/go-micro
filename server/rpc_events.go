package server

import (
	"go-micro.dev/v4/broker"
	log "go-micro.dev/v4/logger"
)

// subscribeServer will subscribe the server to the topic with its own name.
func (s *rpcServer) subscribeServer(config Options) error {
	if s.opts.Router != nil {
		sub, err := s.opts.Broker.Subscribe(config.Name, s.HandleEvent)
		if err != nil {
			return err
		}

		// Save the subscriber
		s.subscriber = sub
	}

	return nil
}

// reSubscribe itterates over subscribers and re-subscribes then
func (s *rpcServer) reSubscribe(config Options) error {
	for sb := range s.subscribers {
		var opts []broker.SubscribeOption
		if queue := sb.Options().Queue; len(queue) > 0 {
			opts = append(opts, broker.Queue(queue))
		}

		if ctx := sb.Options().Context; ctx != nil {
			opts = append(opts, broker.SubscribeContext(ctx))
		}

		if !sb.Options().AutoAck {
			opts = append(opts, broker.DisableAutoAck())
		}

		config.Logger.Logf(log.InfoLevel, "Subscribing to topic: %s", sb.Topic())
		sub, err := config.Broker.Subscribe(sb.Topic(), s.HandleEvent, opts...)
		if err != nil {
			return err
		}

		s.subscribers[sb] = []broker.Subscriber{sub}
	}

	return nil
}

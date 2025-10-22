package server

import (
	"context"
	"fmt"

	"go-micro.dev/v5/broker"
	raw "go-micro.dev/v5/codec/bytes"
	log "go-micro.dev/v5/logger"
	"go-micro.dev/v5/metadata"
	"go-micro.dev/v5/transport/headers"
)

// HandleEvent handles inbound messages to the service directly.
// These events are a result of registering to the topic with the service name.
// TODO: handle requests from an event. We won't send a response.
func (s *rpcServer) HandleEvent(subscriber string) func(e broker.Event) error {
	return func(e broker.Event) error {
		// formatting horrible cruft
		msg := e.Message()

		if msg.Header == nil {
			msg.Header = make(map[string]string)
		}

		contentType, ok := msg.Header["Content-Type"]
		if !ok || len(contentType) == 0 {
			msg.Header["Content-Type"] = DefaultContentType
			contentType = DefaultContentType
		}

		cf, err := s.newCodec(contentType)
		if err != nil {
			return err
		}

		header := make(map[string]string, len(msg.Header))
		for k, v := range msg.Header {
			header[k] = v
		}

		// create context
		ctx := metadata.NewContext(context.Background(), header)

		// TODO: inspect message header for Micro-Service & Micro-Topic
		rpcMsg := &rpcMessage{
			topic:       msg.Header[headers.Message],
			contentType: contentType,
			payload:     &raw.Frame{Data: msg.Body},
			codec:       cf,
			header:      msg.Header,
			body:        msg.Body,
		}

		// if the router is present then execute it
		r := Router(s.router)
		if s.opts.Router != nil {
			// create a wrapped function
			// create a wrapped function
			handler := func(ctx context.Context, msg Message) error {
				return s.opts.Router.ProcessMessage(ctx, subscriber, msg)
			}

			// execute the wrapper for it
			for i := len(s.opts.SubWrappers); i > 0; i-- {
				handler = s.opts.SubWrappers[i-1](handler)
			}

			// set the router
			r = rpcRouter{m: func(ctx context.Context, _ string, msg Message) error {
				return handler(ctx, msg)
			}}
		}

		return r.ProcessMessage(ctx, subscriber, rpcMsg)
	}
}

func (s *rpcServer) NewSubscriber(topic string, sb interface{}, opts ...SubscriberOption) Subscriber {
	return s.router.NewSubscriber(topic, sb, opts...)
}

func (s *rpcServer) Subscribe(sb Subscriber) error {
	s.Lock()
	defer s.Unlock()

	sub, ok := sb.(*subscriber)
	if !ok {
		return fmt.Errorf("invalid subscriber: expected *subscriber")
	}
	if len(sub.handlers) == 0 {
		return fmt.Errorf("invalid subscriber: no handler functions")
	}

	if err := validateSubscriber(sub); err != nil {
		return err
	}

	// append to subscribers
	// subs := s.subscribers[sub.Topic()]
	// subs = append(subs, sub)
	// router.subscribers[sub.Topic()] = subs

	s.subscribers[sb] = nil

	return nil
}

// subscribeServer will subscribe the server to the topic with its own name.
func (s *rpcServer) subscribeServer(config Options) error {
	if s.opts.Router != nil && s.subscriber == nil {
		sub, err := s.opts.Broker.Subscribe(config.Name, s.HandleEvent(config.Name))
		if err != nil {
			return err
		}

		// Save the subscriber
		s.subscriber = sub
	}

	return nil
}

// reSubscribe itterates over subscribers and re-subscribes then.
func (s *rpcServer) reSubscribe(config Options) {
	for sb := range s.subscribers {
		if s.subscribers[sb] != nil {
			continue
		}
		// If we've already created a broker subscription for this topic
		// (from a different Subscriber entry) then don't create another
		// broker.Subscribe. We still need to register the subscriber with
		// the router so it receives dispatched messages.
		var already bool
		for other, subs := range s.subscribers {
			if other.Topic() == sb.Topic() && subs != nil {
				already = true
				break
			}
		}
		if already {
			// register with router only
			if err := s.router.Subscribe(sb); err != nil {
				config.Logger.Logf(log.WarnLevel, "Unable to subscribing to topic: %s, error: %s", sb.Topic(), err)
				continue
			}
			// mark this subscriber as having no broker subscription
			s.subscribers[sb] = nil
			continue
		}
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
		sub, err := config.Broker.Subscribe(sb.Topic(), s.HandleEvent(sb.Topic()), opts...)
		if err != nil {
			config.Logger.Logf(log.WarnLevel, "Unable to subscribing to topic: %s, error: %s", sb.Topic(), err)
			continue
		}
		err = s.router.Subscribe(sb)
		if err != nil {
			config.Logger.Logf(log.WarnLevel, "Unable to subscribing to topic: %s, error: %s", sb.Topic(), err)
			sub.Unsubscribe()
			continue
		}
		s.subscribers[sb] = []broker.Subscriber{sub}
	}
}

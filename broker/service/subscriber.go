package service

import (
	"github.com/micro/go-micro/broker"
	pb "github.com/micro/go-micro/broker/service/proto"
)

type serviceSub struct {
	topic   string
	queue   string
	handler broker.Handler
	stream  pb.Broker_SubscribeService
	closed  chan bool
	options broker.SubscribeOptions
}

type serviceEvent struct {
	topic   string
	message *broker.Message
}

func (s *serviceEvent) Topic() string {
	return s.topic
}

func (s *serviceEvent) Message() *broker.Message {
	return s.message
}

func (s *serviceEvent) Ack() error {
	return nil
}

func (s *serviceSub) run() {
	exit := make(chan bool)
	go func() {
		select {
		case <-exit:
			return
		case <-s.closed:
			s.stream.Close()
		}
	}()

	for {
		// TODO: do not fail silently
		msg, err := s.stream.Recv()
		if err != nil {
			close(exit)
			return
		}
		// TODO: handle error
		s.handler(&serviceEvent{
			topic: s.topic,
			message: &broker.Message{
				Header: msg.Header,
				Body:   msg.Body,
			},
		})
	}
}

func (s *serviceSub) Options() broker.SubscribeOptions {
	return s.options
}

func (s *serviceSub) Topic() string {
	return s.topic
}

func (s *serviceSub) Unsubscribe() error {
	select {
	case <-s.closed:
		return nil
	default:
		close(s.closed)
	}
	return nil
}

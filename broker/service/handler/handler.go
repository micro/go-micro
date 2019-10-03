package handler

import (
	"context"

	"github.com/micro/go-micro/util/log"
	"github.com/micro/go-micro/broker"
	pb "github.com/micro/go-micro/broker/service/proto"
	"github.com/micro/go-micro/errors"
)

type Broker struct {
	Broker broker.Broker
}

func (b *Broker) Publish(ctx context.Context, req *pb.PublishRequest, rsp *pb.Empty) error {
	log.Debugf("Publishing message to %s topic", req.Topic)
	err := b.Broker.Publish(req.Topic, &broker.Message{
		Header: req.Message.Header,
		Body:   req.Message.Body,
	})
	if err != nil {
		return errors.InternalServerError("go.micro.broker", err.Error())
	}
	return nil
}

func (b *Broker) Subscribe(ctx context.Context, req *pb.SubscribeRequest, stream pb.Broker_SubscribeStream) error {
	errChan := make(chan error, 1)

	// message handler to stream back messages from broker
	handler := func(p broker.Event) error {
		if err := stream.Send(&pb.Message{
			Header: p.Message().Header,
			Body:   p.Message().Body,
		}); err != nil {
			select {
			case errChan <- err:
				return err
			default:
				return err
			}
		}
		return nil
	}

	log.Debugf("Subscribing to %s topic", req.Topic)
	sub, err := b.Broker.Subscribe(req.Topic, handler, broker.Queue(req.Queue))
	if err != nil {
		return errors.InternalServerError("go.micro.broker", err.Error())
	}
	defer sub.Unsubscribe()

	select {
	case <-ctx.Done():
		return nil
	case <-errChan:
		return err
	}
}

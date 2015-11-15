package googlepubsub

import (
	"time"

	"github.com/piemapping/go-micro/broker"
	"golang.org/x/net/context"

	log "github.com/golang/glog"
	"google.golang.org/cloud/pubsub"
)

type pullsub struct {
	ctx        context.Context
	name       string
	topic      string
	batchSize  int
	handlers   int
	stopped    chan bool
	msgHandler broker.Handler
}

func (ps *pullsub) Topic() string {
	return ps.topic
}

func (ps *pullsub) Name() string {
	return ps.name
}

func newHandlerFromSubscriber(sub *pullsub, handlerFunc broker.HandlerFunc) broker.Handler {
	return &pubsubHandler{
		ctx:              sub.ctx,
		subscriptionName: sub.name,
		handlerFunc:      handlerFunc,
	}
}

func (ps *pullsub) SetHandlerFunc(h broker.HandlerFunc, concurrency int) {
	ps.msgHandler = newHandlerFromSubscriber(ps, h)
	ps.handlers = concurrency
}

func (ps *pullsub) init() error {
	exists, err := pubsub.SubExists(ps.ctx, ps.name)
	if err != nil {
		return err
	}

	if !exists {
		return pubsub.CreateSub(ps.ctx, ps.name, ps.topic, time.Duration(0), "")
	}

	return nil
}

func (ps *pullsub) Unsubscribe() error {
	if err := pubsub.DeleteSub(ps.ctx, ps.name); err != nil {
		return err
	}

	close(ps.stopped)
	return nil
}

func (ps *pullsub) Subscribe() error {
	err := ps.init()
	if err != nil {
		return err
	}

	msgChan := make(chan *broker.Message, 500)

	for i := 0; i < ps.handlers; i++ {
		handlerNb := i
		go func() {
			log.Infof("[GooglePubSub handler] Started handler number %d", handlerNb)
			for {
				select {
				case msg := <-msgChan:
					if err := ps.msgHandler.Handle(msg); err == nil {
						// If the handling was successful we will attempt to acknowledge the message
						if err := ps.msgHandler.Ack(msg); err != nil {
							log.Errorf("[GooglePubSub handler] Unable to ack message back [message-id=%s]", getMsgAckID(msg))
						}
					}
				case <-ps.stopped:
					log.Infof("[GooglePubSub handler] Closing handler [handler-id=%d]", handlerNb)
					return
				}
			}
		}()
	}

	// Start a separate goroutine to continuously pull
	go func() {
		for {
			select {
			case <-ps.stopped:
				log.Info("[GooglePubSub] Subscription closing")
				return
			default:
				msgs, err := pubsub.PullWait(ps.ctx, ps.name, ps.batchSize)
				if err != nil {
					log.Errorf("[GooglePubSub] Unable to pull messages [subscription-name=%s, topic=%s]: %v", ps.name, ps.topic, err)
					// Introduce artificial sleep - may need to do some kind of exponential back-off
					time.Sleep(3 * time.Second)
					continue
				}

				for _, msg := range msgs {
					bmsg := pubsubMsgToBrokerMsg(msg)

					// Attempt to send message over to processing channel - if the queue is full we will drop the message
					// But it will be requeued at a later stage in the future
					select {
					case msgChan <- bmsg:
					default:
						log.Errorf("[GooglePubSub] Queue is full, unable to process message [message-id=%s]", msg.ID)
					}
				}
			}
		}
	}()

	log.Info("[GooglePubSub] Subscription started [name=%s, topic=%s]", ps.name, ps.topic)
	return nil
}

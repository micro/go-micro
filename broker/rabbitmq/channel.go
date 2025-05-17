package rabbitmq

//
// All credit to Mondo
//

import (
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type rabbitMQChannel struct {
	uuid           string
	connection     *amqp.Connection
	channel        *amqp.Channel
	confirmPublish chan amqp.Confirmation
	mtx            sync.Mutex
}

func newRabbitChannel(conn *amqp.Connection, prefetchCount int, prefetchGlobal bool, confirmPublish bool) (*rabbitMQChannel, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	rabbitCh := &rabbitMQChannel{
		uuid:       id.String(),
		connection: conn,
	}
	if err := rabbitCh.Connect(prefetchCount, prefetchGlobal, confirmPublish); err != nil {
		return nil, err
	}
	return rabbitCh, nil
}

func (r *rabbitMQChannel) Connect(prefetchCount int, prefetchGlobal bool, confirmPublish bool) error {
	var err error
	r.channel, err = r.connection.Channel()
	if err != nil {
		return err
	}

	err = r.channel.Qos(prefetchCount, 0, prefetchGlobal)
	if err != nil {
		return err
	}

	if confirmPublish {
		r.confirmPublish = r.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

		err = r.channel.Confirm(false)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *rabbitMQChannel) Close() error {
	if r.channel == nil {
		return errors.New("Channel is nil")
	}
	return r.channel.Close()
}

func (r *rabbitMQChannel) Publish(exchange, key string, message amqp.Publishing) error {
	if r.channel == nil {
		return errors.New("Channel is nil")
	}

	if r.confirmPublish != nil {
		r.mtx.Lock()
		defer r.mtx.Unlock()
	}

	err := r.channel.Publish(exchange, key, false, false, message)
	if err != nil {
		return err
	}

	if r.confirmPublish != nil {
		confirmation, ok := <-r.confirmPublish
		if !ok {
			return errors.New("Channel closed before could receive confirmation of publish")
		}

		if !confirmation.Ack {
			return errors.New("Could not publish message, received nack from broker on confirmation")
		}
	}

	return nil
}

func (r *rabbitMQChannel) DeclareExchange(ex Exchange) error {
	return r.channel.ExchangeDeclare(
		ex.Name,         // name
		string(ex.Type), // kind
		ex.Durable,      // durable
		false,           // autoDelete
		false,           // internal
		false,           // noWait
		nil,             // args
	)
}

func (r *rabbitMQChannel) DeclareDurableExchange(ex Exchange) error {
	return r.channel.ExchangeDeclare(
		ex.Name,         // name
		string(ex.Type), // kind
		true,            // durable
		false,           // autoDelete
		false,           // internal
		false,           // noWait
		nil,             // args
	)
}

func (r *rabbitMQChannel) DeclareQueue(queue string, args amqp.Table) error {
	_, err := r.channel.QueueDeclare(
		queue, // name
		false, // durable
		true,  // autoDelete
		false, // exclusive
		false, // noWait
		args,  // args
	)
	return err
}

func (r *rabbitMQChannel) DeclareDurableQueue(queue string, args amqp.Table) error {
	_, err := r.channel.QueueDeclare(
		queue, // name
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		args,  // args
	)
	return err
}

func (r *rabbitMQChannel) DeclareReplyQueue(queue string) error {
	_, err := r.channel.QueueDeclare(
		queue, // name
		false, // durable
		true,  // autoDelete
		true,  // exclusive
		false, // noWait
		nil,   // args
	)
	return err
}

func (r *rabbitMQChannel) ConsumeQueue(queue string, autoAck bool) (<-chan amqp.Delivery, error) {
	return r.channel.Consume(
		queue,   // queue
		r.uuid,  // consumer
		autoAck, // autoAck
		false,   // exclusive
		false,   // nolocal
		false,   // nowait
		nil,     // args
	)
}

func (r *rabbitMQChannel) BindQueue(queue, key, exchange string, args amqp.Table) error {
	return r.channel.QueueBind(
		queue,    // name
		key,      // key
		exchange, // exchange
		false,    // noWait
		args,     // args
	)
}

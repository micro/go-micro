package rabbitmq

//
// All credit to Mondo
//

import (
	"errors"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type rabbitMQChannel struct {
	uuid       string
	connection *amqp.Connection
	channel    *amqp.Channel
}

func newRabbitChannel(conn *amqp.Connection) (*rabbitMQChannel, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	rabbitCh := &rabbitMQChannel{
		uuid:       id.String(),
		connection: conn,
	}
	if err := rabbitCh.Connect(); err != nil {
		return nil, err
	}
	return rabbitCh, nil

}

func (r *rabbitMQChannel) Connect() error {
	var err error
	r.channel, err = r.connection.Channel()
	if err != nil {
		return err
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
	return r.channel.Publish(exchange, key, false, false, message)
}

func (r *rabbitMQChannel) DeclareExchange(exchange string) error {
	return r.channel.ExchangeDeclare(
		exchange, // name
		"topic",  // kind
		false,    // durable
		false,    // autoDelete
		false,    // internal
		false,    // noWait
		nil,      // args
	)
}

func (r *rabbitMQChannel) DeclareQueue(queue string) error {
	_, err := r.channel.QueueDeclare(
		queue, // name
		false, // durable
		true,  // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	return err
}

func (r *rabbitMQChannel) DeclareDurableQueue(queue string) error {
	_, err := r.channel.QueueDeclare(
		queue, // name
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
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

func (r *rabbitMQChannel) ConsumeQueue(queue string) (<-chan amqp.Delivery, error) {
	return r.channel.Consume(
		queue,  // queue
		r.uuid, // consumer
		true,   // autoAck
		false,  // exclusive
		false,  // nolocal
		false,  // nowait
		nil,    // args
	)
}

func (r *rabbitMQChannel) BindQueue(queue, exchange string) error {
	return r.channel.QueueBind(
		queue,    // name
		queue,    // key
		exchange, // exchange
		false,    // noWait
		nil,      // args
	)
}

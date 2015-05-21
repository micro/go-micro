package transport

//
// All credit to Mondo
//

import (
	"errors"

	"github.com/nu7hatch/gouuid"
	"github.com/streadway/amqp"
)

type RabbitChannel struct {
	uuid       string
	connection *amqp.Connection
	channel    *amqp.Channel
}

func NewRabbitChannel(conn *amqp.Connection) (*RabbitChannel, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	rabbitCh := &RabbitChannel{
		uuid:       id.String(),
		connection: conn,
	}
	if err := rabbitCh.Connect(); err != nil {
		return nil, err
	}
	return rabbitCh, nil

}

func (r *RabbitChannel) Connect() error {
	var err error
	r.channel, err = r.connection.Channel()
	if err != nil {
		return err
	}
	return nil
}

func (r *RabbitChannel) Close() error {
	if r.channel == nil {
		return errors.New("Channel is nil")
	}
	return r.channel.Close()
}

func (r *RabbitChannel) Publish(exchange, key string, message amqp.Publishing) error {
	if r.channel == nil {
		return errors.New("Channel is nil")
	}
	return r.channel.Publish(exchange, key, false, false, message)
}

func (r *RabbitChannel) DeclareExchange(exchange string) error {
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

func (r *RabbitChannel) DeclareQueue(queue string) error {
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

func (r *RabbitChannel) DeclareDurableQueue(queue string) error {
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

func (r *RabbitChannel) DeclareReplyQueue(queue string) error {
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

func (r *RabbitChannel) ConsumeQueue(queue string) (<-chan amqp.Delivery, error) {
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

func (r *RabbitChannel) BindQueue(queue, exchange string) error {
	return r.channel.QueueBind(
		queue,    // name
		queue,    // key
		exchange, // exchange
		false,    // noWait
		nil,      // args
	)
}

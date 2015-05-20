package transport

//
// All credit to Mondo
// https://github.com/mondough/typhon
//

import (
	"sync"
	"time"

	"github.com/streadway/amqp"
)

var (
	DefaultExchange  = "micro"
	DefaultRabbitURL = "amqp://guest:guest@127.0.0.1:5672"
)

type RabbitConnection struct {
	Connection      *amqp.Connection
	Channel         *RabbitChannel
	ExchangeChannel *RabbitChannel
	notify          chan bool
	exchange        string
	url             string

	connected bool

	mtx       sync.Mutex
	closeChan chan struct{}
	closed    bool
}

func (r *RabbitConnection) Init() chan bool {
	go r.Connect(r.notify)
	return r.notify
}

func (r *RabbitConnection) Connect(connected chan bool) {
	for {
		if err := r.tryToConnect(); err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		connected <- true
		r.connected = true
		notifyClose := make(chan *amqp.Error)
		r.Connection.NotifyClose(notifyClose)

		// Block until we get disconnected, or shut down
		select {
		case <-notifyClose:
			// Spin around and reconnect
			r.connected = false
		case <-r.closeChan:
			// Shut down connection
			if err := r.Connection.Close(); err != nil {
			}
			r.connected = false
			return
		}
	}
}

func (r *RabbitConnection) IsConnected() bool {
	return r.connected
}

func (r *RabbitConnection) Close() {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.closed {
		return
	}

	close(r.closeChan)
	r.closed = true
}

func (r *RabbitConnection) tryToConnect() error {
	var err error
	r.Connection, err = amqp.Dial(r.url)
	if err != nil {
		return err
	}
	r.Channel, err = NewRabbitChannel(r.Connection)
	if err != nil {
		return err
	}
	r.Channel.DeclareExchange(r.exchange)
	r.ExchangeChannel, err = NewRabbitChannel(r.Connection)
	if err != nil {
		return err
	}
	return nil
}

func (r *RabbitConnection) Consume(serverName string) (<-chan amqp.Delivery, error) {
	consumerChannel, err := NewRabbitChannel(r.Connection)
	if err != nil {
		return nil, err
	}

	err = consumerChannel.DeclareQueue(serverName)
	if err != nil {
		return nil, err
	}

	deliveries, err := consumerChannel.ConsumeQueue(serverName)
	if err != nil {
		return nil, err
	}

	err = consumerChannel.BindQueue(serverName, r.exchange)
	if err != nil {
		return nil, err
	}

	return deliveries, nil
}

func (r *RabbitConnection) Publish(exchange, routingKey string, msg amqp.Publishing) error {
	return r.ExchangeChannel.Publish(exchange, routingKey, msg)
}

func NewRabbitConnection(exchange, url string) *RabbitConnection {
	if len(url) == 0 {
		url = DefaultRabbitURL
	}

	if len(exchange) == 0 {
		exchange = DefaultExchange
	}

	return &RabbitConnection{
		exchange:  DefaultExchange,
		url:       DefaultRabbitURL,
		notify:    make(chan bool, 1),
		closeChan: make(chan struct{}),
	}
}

package rabbitmq

//
// All credit to Mondo
//

import (
	"crypto/tls"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"go-micro.dev/v5/logger"
)

type MQExchangeType string

const (
	ExchangeTypeFanout MQExchangeType = "fanout"
	ExchangeTypeTopic                 = "topic"
	ExchangeTypeDirect                = "direct"
)

var (
	DefaultExchange = Exchange{
		Name: "micro",
		Type: ExchangeTypeTopic,
	}
	DefaultRabbitURL       = "amqp://guest:guest@127.0.0.1:5672"
	DefaultPrefetchCount   = 0
	DefaultPrefetchGlobal  = false
	DefaultRequeueOnError  = false
	DefaultConfirmPublish  = false
	DefaultWithoutExchange = false

	// The amqp library does not seem to set these when using amqp.DialConfig
	// (even though it says so in the comments) so we set them manually to make
	// sure to not brake any existing functionality.
	defaultHeartbeat = 10 * time.Second
	defaultLocale    = "en_US"

	defaultAmqpConfig = amqp.Config{
		Heartbeat: defaultHeartbeat,
		Locale:    defaultLocale,
	}

	dial       = amqp.Dial
	dialTLS    = amqp.DialTLS
	dialConfig = amqp.DialConfig
)

type rabbitMQConn struct {
	Connection      *amqp.Connection
	Channel         *rabbitMQChannel
	ExchangeChannel *rabbitMQChannel
	exchange        Exchange
	withoutExchange bool
	url             string
	prefetchCount   int
	prefetchGlobal  bool
	confirmPublish  bool

	sync.Mutex
	connected bool
	close     chan bool

	waitConnection chan struct{}

	logger logger.Logger
}

// Exchange is the rabbitmq exchange.
type Exchange struct {
	// Name of the exchange
	Name string
	// Type of the exchange
	Type MQExchangeType
	// Whether its persistent
	Durable bool
}

func newRabbitMQConn(ex Exchange, urls []string, prefetchCount int, prefetchGlobal bool, confirmPublish bool, withoutExchange bool, logger logger.Logger) *rabbitMQConn {
	var url string

	if len(urls) > 0 && regexp.MustCompile("^amqp(s)?://.*").MatchString(urls[0]) {
		url = urls[0]
	} else {
		url = DefaultRabbitURL
	}

	ret := &rabbitMQConn{
		exchange:        ex,
		url:             url,
		withoutExchange: withoutExchange,
		prefetchCount:   prefetchCount,
		prefetchGlobal:  prefetchGlobal,
		confirmPublish:  confirmPublish,
		close:           make(chan bool),
		waitConnection:  make(chan struct{}),
		logger:          logger,
	}
	// its bad case of nil == waitConnection, so close it at start
	close(ret.waitConnection)
	return ret
}

func (r *rabbitMQConn) connect(secure bool, config *amqp.Config) error {
	// try connect
	if err := r.tryConnect(secure, config); err != nil {
		return err
	}

	// connected
	r.Lock()
	r.connected = true
	r.Unlock()

	// create reconnect loop
	go r.reconnect(secure, config)
	return nil
}

func (r *rabbitMQConn) reconnect(secure bool, config *amqp.Config) {
	// skip first connect
	var connect bool

	for {
		if connect {
			// try reconnect
			if err := r.tryConnect(secure, config); err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			// connected
			r.Lock()
			r.connected = true
			r.Unlock()
			// unblock resubscribe cycle - close channel
			//at this point channel is created and unclosed - close it without any additional checks
			close(r.waitConnection)
		}

		connect = true
		notifyClose := make(chan *amqp.Error)
		r.Connection.NotifyClose(notifyClose)
		chanNotifyClose := make(chan *amqp.Error)
		var channel *amqp.Channel
		if !r.withoutExchange {
			channel = r.ExchangeChannel.channel
		} else {
			channel = r.Channel.channel
		}
		channel.NotifyClose(chanNotifyClose)
		// To avoid deadlocks it is necessary to consume the messages from all channels.
		for notifyClose != nil || chanNotifyClose != nil {
			// block until closed
			select {
			case err := <-chanNotifyClose:
				r.logger.Log(logger.ErrorLevel, err)
				// block all resubscribe attempt - they are useless because there is no connection to rabbitmq
				// create channel 'waitConnection' (at this point channel is nil or closed, create it without unnecessary checks)
				r.Lock()
				r.connected = false
				r.waitConnection = make(chan struct{})
				r.Unlock()
				chanNotifyClose = nil
			case err := <-notifyClose:
				r.logger.Log(logger.ErrorLevel, err)
				// block all resubscribe attempt - they are useless because there is no connection to rabbitmq
				// create channel 'waitConnection' (at this point channel is nil or closed, create it without unnecessary checks)
				r.Lock()
				r.connected = false
				r.waitConnection = make(chan struct{})
				r.Unlock()
				notifyClose = nil
			case <-r.close:
				return
			}
		}
	}
}

func (r *rabbitMQConn) Connect(secure bool, config *amqp.Config) error {
	r.Lock()

	// already connected
	if r.connected {
		r.Unlock()
		return nil
	}

	// check it was closed
	select {
	case <-r.close:
		r.close = make(chan bool)
	default:
		// no op
		// new conn
	}

	r.Unlock()

	return r.connect(secure, config)
}

func (r *rabbitMQConn) Close() error {
	r.Lock()
	defer r.Unlock()

	select {
	case <-r.close:
		return nil
	default:
		close(r.close)
		r.connected = false
	}

	return r.Connection.Close()
}

func (r *rabbitMQConn) tryConnect(secure bool, config *amqp.Config) error {
	var err error

	if config == nil {
		config = &defaultAmqpConfig
	}

	url := r.url

	if secure || config.TLSClientConfig != nil || strings.HasPrefix(r.url, "amqps://") {
		if config.TLSClientConfig == nil {
			config.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}

		url = strings.Replace(r.url, "amqp://", "amqps://", 1)
	}

	r.Connection, err = dialConfig(url, *config)

	if err != nil {
		return err
	}

	if r.Channel, err = newRabbitChannel(r.Connection, r.prefetchCount, r.prefetchGlobal, r.confirmPublish); err != nil {
		return err
	}

	if !r.withoutExchange {
		if r.exchange.Durable {
			r.Channel.DeclareDurableExchange(r.exchange)
		} else {
			r.Channel.DeclareExchange(r.exchange)
		}
		r.ExchangeChannel, err = newRabbitChannel(r.Connection, r.prefetchCount, r.prefetchGlobal, r.confirmPublish)
	}
	return err
}

func (r *rabbitMQConn) Consume(queue, key string, headers amqp.Table, qArgs amqp.Table, autoAck, durableQueue bool) (*rabbitMQChannel, <-chan amqp.Delivery, error) {
	consumerChannel, err := newRabbitChannel(r.Connection, r.prefetchCount, r.prefetchGlobal, r.confirmPublish)
	if err != nil {
		return nil, nil, err
	}

	if durableQueue {
		err = consumerChannel.DeclareDurableQueue(queue, qArgs)
	} else {
		err = consumerChannel.DeclareQueue(queue, qArgs)
	}

	if err != nil {
		return nil, nil, err
	}

	deliveries, err := consumerChannel.ConsumeQueue(queue, autoAck)
	if err != nil {
		return nil, nil, err
	}

	if !r.withoutExchange {
		err = consumerChannel.BindQueue(queue, key, r.exchange.Name, headers)
		if err != nil {
			return nil, nil, err
		}
	}

	return consumerChannel, deliveries, nil
}

func (r *rabbitMQConn) Publish(exchange, key string, msg amqp.Publishing) error {
	if r.withoutExchange {
		return r.Channel.Publish("", key, msg)
	}
	return r.ExchangeChannel.Publish(exchange, key, msg)
}

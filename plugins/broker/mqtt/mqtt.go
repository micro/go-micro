// Package mqtt provides an mqtt broker
package mqtt

/*
	MQTT is a go-micro Broker for the MQTT protocol.
	This can be integrated with any broker that supports MQTT,
	including Mosquito and AWS IoT.

	Note: Because of the way the MQTT library works, when you
	unsubscribe from a topic it will unsubscribe all subscribers.
	TODO: Perhaps create a unique client per subscription.
	Becomes slightly more difficult to track a disconnect.

*/

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/codec/json"
	"github.com/asim/go-micro/v3/cmd"
	log "github.com/asim/go-micro/v3/logger"
)

type mqttBroker struct {
	addrs  []string
	opts   broker.Options
	client mqtt.Client
}

func init() {
	cmd.DefaultBrokers["mqtt"] = NewBroker
	rand.Seed(time.Now().UnixNano())
}

func setAddrs(addrs []string) []string {
	cAddrs := make([]string, 0, len(addrs))

	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}

		var scheme string
		var host string
		var port int

		// split on scheme
		parts := strings.Split(addr, "://")

		// no scheme
		if len(parts) < 2 {
			// default tcp scheme
			scheme = "tcp"
			parts = strings.Split(parts[0], ":")
			// got scheme
		} else {
			scheme = parts[0]
			parts = strings.Split(parts[1], ":")
		}

		// no parts
		if len(parts) == 0 {
			continue
		}

		// check scheme
		switch scheme {
		case "tcp", "ssl", "ws":
		default:
			continue
		}

		if len(parts) < 2 {
			// no port
			host = parts[0]

			switch scheme {
			case "tcp":
				port = 1883
			case "ssl":
				port = 8883
			case "ws":
				// support secure port
				port = 80
			default:
				port = 1883
			}
			// got host port
		} else {
			host = parts[0]
			port, _ = strconv.Atoi(parts[1])
		}

		addr = fmt.Sprintf("%s://%s:%d", scheme, host, port)
		cAddrs = append(cAddrs, addr)

	}

	// default an address if we have none
	if len(cAddrs) == 0 {
		cAddrs = []string{"tcp://127.0.0.1:1883"}
	}

	return cAddrs
}

func newClient(addrs []string, opts broker.Options) mqtt.Client {
	// create opts
	cOpts := mqtt.NewClientOptions()
	cOpts.SetClientID(fmt.Sprintf("%d%d", time.Now().UnixNano(), rand.Intn(10)))
	cOpts.SetCleanSession(false)

	// setup tls
	if opts.TLSConfig != nil {
		cOpts.SetTLSConfig(opts.TLSConfig)
	}

	// add brokers
	for _, addr := range addrs {
		cOpts.AddBroker(addr)
	}

	return mqtt.NewClient(cOpts)
}

func newBroker(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		// Default codec
		Codec: json.Marshaler{},
	}

	for _, o := range opts {
		o(&options)
	}

	addrs := setAddrs(options.Addrs)
	client := newClient(addrs, options)

	return &mqttBroker{
		opts:   options,
		client: client,
		addrs:  addrs,
	}
}

func (m *mqttBroker) Options() broker.Options {
	return m.opts
}

func (m *mqttBroker) Address() string {
	return strings.Join(m.addrs, ",")
}

func (m *mqttBroker) Connect() error {
	if m.client.IsConnected() {
		return nil
	}

	if t := m.client.Connect(); t.Wait() && t.Error() != nil {
		return t.Error()
	}

	return nil
}

func (m *mqttBroker) Disconnect() error {
	if !m.client.IsConnected() {
		return nil
	}
	m.client.Disconnect(0)
	return nil
}

func (m *mqttBroker) Init(opts ...broker.Option) error {
	if m.client.IsConnected() {
		return errors.New("cannot init while connected")
	}

	for _, o := range opts {
		o(&m.opts)
	}

	m.addrs = setAddrs(m.opts.Addrs)
	m.client = newClient(m.addrs, m.opts)
	return nil
}

func (m *mqttBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	if !m.client.IsConnected() {
		return errors.New("not connected")
	}

	b, err := m.opts.Codec.Marshal(msg)
	if err != nil {
		return err
	}

	t := m.client.Publish(topic, 1, false, b)
	return t.Error()
}

func (m *mqttBroker) Subscribe(topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if !m.client.IsConnected() {
		return nil, errors.New("not connected")
	}

	var options broker.SubscribeOptions
	for _, o := range opts {
		o(&options)
	}

	t := m.client.Subscribe(topic, 1, func(c mqtt.Client, mq mqtt.Message) {
		var msg broker.Message
		if err := m.opts.Codec.Unmarshal(mq.Payload(), &msg); err != nil {
			log.Error(err)
			return
		}

		p := &mqttPub{topic: topic, msg: &msg}
		if err := h(p); err != nil {
			p.err = err
			log.Error(err)
		}
	})

	if t.Wait() && t.Error() != nil {
		return nil, t.Error()
	}

	return &mqttSub{
		opts:   options,
		client: m.client,
		topic:  topic,
	}, nil
}

func (m *mqttBroker) String() string {
	return "mqtt"
}

func NewBroker(opts ...broker.Option) broker.Broker {
	return newBroker(opts...)
}

package mqtt

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/asim/go-micro/v3/broker"
)

// mqttPub is a broker.Event
type mqttPub struct {
	topic string
	msg   *broker.Message
	err   error
}

// mqttPub is a broker.Subscriber
type mqttSub struct {
	opts   broker.SubscribeOptions
	topic  string
	client mqtt.Client
}

func (m *mqttPub) Ack() error {
	return nil
}

func (m *mqttPub) Error() error {
	return m.err
}

func (m *mqttPub) Topic() string {
	return m.topic
}

func (m *mqttPub) Message() *broker.Message {
	return m.msg
}

func (m *mqttSub) Options() broker.SubscribeOptions {
	return m.opts
}

func (m *mqttSub) Topic() string {
	return m.topic
}

func (m *mqttSub) Unsubscribe() error {
	t := m.client.Unsubscribe(m.topic)
	return t.Error()
}

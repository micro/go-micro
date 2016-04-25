package mqtt

import (
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/micro/go-micro/broker"
)

// mqttPub is a broker.Publication
type mqttPub struct {
	topic string
	msg   *broker.Message
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

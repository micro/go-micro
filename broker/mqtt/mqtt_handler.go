package mqtt

import (
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/micro/go-micro/broker"
)

// mqttPub is a broker.Publication
type mqttPub struct {
	msg mqtt.Message
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
	return m.msg.Topic()
}

func (m *mqttPub) Message() *broker.Message {
	// TODO: Support encoding to preserve headers
	return &broker.Message{
		Body: m.msg.Payload(),
	}
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

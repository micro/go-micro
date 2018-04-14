package client

type message struct {
	topic       string
	contentType string
	payload     interface{}
}

func newMessage(topic string, payload interface{}, contentType string) Message {
	return &message{
		payload:     payload,
		topic:       topic,
		contentType: contentType,
	}
}

func (m *message) ContentType() string {
	return m.contentType
}

func (m *message) Topic() string {
	return m.topic
}

func (m *message) Payload() interface{} {
	return m.payload
}

package client

type rpcPublication struct {
	topic       string
	contentType string
	message     interface{}
}

func newRpcPublication(topic string, message interface{}, contentType string) Publication {
	return &rpcPublication{
		message:     message,
		topic:       topic,
		contentType: contentType,
	}
}

func (r *rpcPublication) ContentType() string {
	return r.contentType
}

func (r *rpcPublication) Topic() string {
	return r.topic
}

func (r *rpcPublication) Message() interface{} {
	return r.message
}

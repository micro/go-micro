package mq

import (
	"go-micro.dev/v4/api/client"
)

type Mq interface {
	Publish(*PublishRequest) (*PublishResponse, error)
	Subscribe(*SubscribeRequest) (*SubscribeResponseStream, error)
}

func NewMqService(token string) *MqService {
	return &MqService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type MqService struct {
	client *client.Client
}

// Publish a message. Specify a topic to group messages for a specific topic.
func (t *MqService) Publish(request *PublishRequest) (*PublishResponse, error) {

	rsp := &PublishResponse{}
	return rsp, t.client.Call("mq", "Publish", request, rsp)

}

// Subscribe to messages for a given topic.
func (t *MqService) Subscribe(request *SubscribeRequest) (*SubscribeResponseStream, error) {
	stream, err := t.client.Stream("mq", "Subscribe", request)
	if err != nil {
		return nil, err
	}
	return &SubscribeResponseStream{
		stream: stream,
	}, nil

}

type SubscribeResponseStream struct {
	stream *client.Stream
}

func (t *SubscribeResponseStream) Recv() (*SubscribeResponse, error) {
	var rsp SubscribeResponse
	if err := t.stream.Recv(&rsp); err != nil {
		return nil, err
	}
	return &rsp, nil
}

type PublishRequest struct {
	// The json message to publish
	Message map[string]interface{} `json:"message"`
	// The topic to publish to
	Topic string `json:"topic"`
}

type PublishResponse struct {
}

type SubscribeRequest struct {
	// The topic to subscribe to
	Topic string `json:"topic"`
}

type SubscribeResponse struct {
	// The next json message on the topic
	Message map[string]interface{} `json:"message"`
	// The topic subscribed to
	Topic string `json:"topic"`
}

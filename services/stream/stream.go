package stream

import (
	"github.com/m3o/m3o-go/client"
)

func NewStreamService(token string) *StreamService {
	return &StreamService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type StreamService struct {
	client *client.Client
}

// Publish a message to the stream. Specify a topic to group messages for a specific topic.
func (t *StreamService) Publish(request *PublishRequest) (*PublishResponse, error) {
	rsp := &PublishResponse{}
	return rsp, t.client.Call("stream", "Publish", request, rsp)
}

// Subscribe to messages for a given topic.
func (t *StreamService) Subscribe(request *SubscribeRequest) (*SubscribeResponse, error) {
	rsp := &SubscribeResponse{}
	return rsp, t.client.Call("stream", "Subscribe", request, rsp)
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

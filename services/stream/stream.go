package stream

import (
	"go-micro.dev/v4/api/client"
)

type Stream interface {
	CreateChannel(*CreateChannelRequest) (*CreateChannelResponse, error)
	ListChannels(*ListChannelsRequest) (*ListChannelsResponse, error)
	ListMessages(*ListMessagesRequest) (*ListMessagesResponse, error)
	SendMessage(*SendMessageRequest) (*SendMessageResponse, error)
}

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

// Create a channel with a given name and description. Channels are created automatically but
// this allows you to specify a description that's persisted for the lifetime of the channel.
func (t *StreamService) CreateChannel(request *CreateChannelRequest) (*CreateChannelResponse, error) {

	rsp := &CreateChannelResponse{}
	return rsp, t.client.Call("stream", "CreateChannel", request, rsp)

}

// List all the active channels
func (t *StreamService) ListChannels(request *ListChannelsRequest) (*ListChannelsResponse, error) {

	rsp := &ListChannelsResponse{}
	return rsp, t.client.Call("stream", "ListChannels", request, rsp)

}

// List messages for a given channel
func (t *StreamService) ListMessages(request *ListMessagesRequest) (*ListMessagesResponse, error) {

	rsp := &ListMessagesResponse{}
	return rsp, t.client.Call("stream", "ListMessages", request, rsp)

}

// Send a message to the stream.
func (t *StreamService) SendMessage(request *SendMessageRequest) (*SendMessageResponse, error) {

	rsp := &SendMessageResponse{}
	return rsp, t.client.Call("stream", "SendMessage", request, rsp)

}

type Channel struct {
	// description for the channel
	Description string `json:"description"`
	// last activity time
	LastActive string `json:"last_active"`
	// name of the channel
	Name string `json:"name"`
}

type CreateChannelRequest struct {
	// description for the channel
	Description string `json:"description"`
	// name of the channel
	Name string `json:"name"`
}

type CreateChannelResponse struct {
}

type ListChannelsRequest struct {
}

type ListChannelsResponse struct {
	Channels []Channel `json:"channels"`
}

type ListMessagesRequest struct {
	// The channel to subscribe to
	Channel string `json:"channel"`
	// number of message to return
	Limit int32 `json:"limit"`
}

type ListMessagesResponse struct {
	// The channel subscribed to
	Channel string `json:"channel"`
	// Messages are chronological order
	Messages []Message `json:"messages"`
}

type Message struct {
	// the channel name
	Channel string `json:"channel"`
	// id of the message
	Id string `json:"id"`
	// the associated metadata
	Metadata map[string]string `json:"metadata"`
	// text of the message
	Text string `json:"text"`
	// time of message creation
	Timestamp string `json:"timestamp"`
}

type SendMessageRequest struct {
	// The channel to send to
	Channel string `json:"channel"`
	// The message text to send
	Text string `json:"text"`
}

type SendMessageResponse struct {
}

package helloworld

import (
	"go-micro.dev/v4/api/client"
)

type Helloworld interface {
	Call(*CallRequest) (*CallResponse, error)
	Stream(*StreamRequest) (*StreamResponseStream, error)
}

func NewHelloworldService(token string) *HelloworldService {
	return &HelloworldService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type HelloworldService struct {
	client *client.Client
}

// Call returns a personalised "Hello $name" response
func (t *HelloworldService) Call(request *CallRequest) (*CallResponse, error) {

	rsp := &CallResponse{}
	return rsp, t.client.Call("helloworld", "Call", request, rsp)

}

// Stream returns a stream of "Hello $name" responses
func (t *HelloworldService) Stream(request *StreamRequest) (*StreamResponseStream, error) {
	stream, err := t.client.Stream("helloworld", "Stream", request)
	if err != nil {
		return nil, err
	}
	return &StreamResponseStream{
		stream: stream,
	}, nil

}

type StreamResponseStream struct {
	stream *client.Stream
}

func (t *StreamResponseStream) Recv() (*StreamResponse, error) {
	var rsp StreamResponse
	if err := t.stream.Recv(&rsp); err != nil {
		return nil, err
	}
	return &rsp, nil
}

type CallRequest struct {
	Name string `json:"name"`
}

type CallResponse struct {
	Message string `json:"message"`
}

type StreamRequest struct {
	// the number of messages to send back
	Messages int64  `json:"messages,string"`
	Name     string `json:"name"`
}

type StreamResponse struct {
	Message string `json:"message"`
}

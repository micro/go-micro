package sms

import (
	"go-micro.dev/v4/api/client"
)

type Sms interface {
	Send(*SendRequest) (*SendResponse, error)
}

func NewSmsService(token string) *SmsService {
	return &SmsService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type SmsService struct {
	client *client.Client
}

// Send an SMS.
func (t *SmsService) Send(request *SendRequest) (*SendResponse, error) {

	rsp := &SendResponse{}
	return rsp, t.client.Call("sms", "Send", request, rsp)

}

type SendRequest struct {
	// who is the message from? The message will be suffixed with "Sent from <from>"
	From string `json:"from"`
	// the main body of the message to send
	Message string `json:"message"`
	// the destination phone number including the international dialling code (e.g. +44)
	To string `json:"to"`
}

type SendResponse struct {
	// any additional info
	Info string `json:"info"`
	// will return "ok" if successful
	Status string `json:"status"`
}

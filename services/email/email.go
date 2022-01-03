package email

import (
	"go-micro.dev/v4/api/client"
)

type Email interface {
	Send(*SendRequest) (*SendResponse, error)
}

func NewEmailService(token string) *EmailService {
	return &EmailService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type EmailService struct {
	client *client.Client
}

// Send an email by passing in from, to, subject, and a text or html body
func (t *EmailService) Send(request *SendRequest) (*SendResponse, error) {

	rsp := &SendResponse{}
	return rsp, t.client.Call("email", "Send", request, rsp)

}

type SendRequest struct {
	// the display name of the sender
	From string `json:"from"`
	// the html body
	HtmlBody string `json:"html_body"`
	// an optional reply to email address
	ReplyTo string `json:"reply_to"`
	// the email subject
	Subject string `json:"subject"`
	// the text body
	TextBody string `json:"text_body"`
	// the email address of the recipient
	To string `json:"to"`
}

type SendResponse struct {
}

package email

import (
	"github.com/m3o/m3o-go/client"
)

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
	HtmlBody string `json:"htmlBody"`
	// an optional reply to email address
	ReplyTo string `json:"replyTo"`
	// the email subject
	Subject string `json:"subject"`
	// the text body
	TextBody string `json:"textBody"`
	// the email address of the recipient
	To string `json:"to"`
}

type SendResponse struct {
}

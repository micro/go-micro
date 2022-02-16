package email

import (
	"go-micro.dev/v4/api/client"
)

type Email interface {
	Parse(*ParseRequest) (*ParseResponse, error)
	Send(*SendRequest) (*SendResponse, error)
	Validate(*ValidateRequest) (*ValidateResponse, error)
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

// Parse an RFC5322 address e.g "Joe Blogs <joe@example.com>"
func (t *EmailService) Parse(request *ParseRequest) (*ParseResponse, error) {

	rsp := &ParseResponse{}
	return rsp, t.client.Call("email", "Parse", request, rsp)

}

// Send an email by passing in from, to, subject, and a text or html body
func (t *EmailService) Send(request *SendRequest) (*SendResponse, error) {

	rsp := &SendResponse{}
	return rsp, t.client.Call("email", "Send", request, rsp)

}

// Validate an email address format
func (t *EmailService) Validate(request *ValidateRequest) (*ValidateResponse, error) {

	rsp := &ValidateResponse{}
	return rsp, t.client.Call("email", "Validate", request, rsp)

}

type ParseRequest struct {
	// The address to parse. Can be of the format "Joe Blogs <joe@example.com>" or "joe@example.com"
	Address string `json:"address"`
}

type ParseResponse struct {
	// the email address
	Address string `json:"address"`
	// associated name e.g Joe Blogs
	Name string `json:"name"`
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

type ValidateRequest struct {
	Address string `json:"address"`
}

type ValidateResponse struct {
	IsValid bool `json:"is_valid"`
}

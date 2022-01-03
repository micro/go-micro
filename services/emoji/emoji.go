package emoji

import (
	"go-micro.dev/v4/api/client"
)

type Emoji interface {
	Find(*FindRequest) (*FindResponse, error)
	Flag(*FlagRequest) (*FlagResponse, error)
	Print(*PrintRequest) (*PrintResponse, error)
	Send(*SendRequest) (*SendResponse, error)
}

func NewEmojiService(token string) *EmojiService {
	return &EmojiService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type EmojiService struct {
	client *client.Client
}

// Find an emoji by its alias e.g :beer:
func (t *EmojiService) Find(request *FindRequest) (*FindResponse, error) {

	rsp := &FindResponse{}
	return rsp, t.client.Call("emoji", "Find", request, rsp)

}

// Get the flag for a country. Requires country code e.g GB for great britain
func (t *EmojiService) Flag(request *FlagRequest) (*FlagResponse, error) {

	rsp := &FlagResponse{}
	return rsp, t.client.Call("emoji", "Flag", request, rsp)

}

// Print text and renders the emojis with aliases e.g
// let's grab a :beer: becomes let's grab a 🍺
func (t *EmojiService) Print(request *PrintRequest) (*PrintResponse, error) {

	rsp := &PrintResponse{}
	return rsp, t.client.Call("emoji", "Print", request, rsp)

}

// Send an emoji to anyone via SMS. Messages are sent in the form '<message> Sent from <from>'
func (t *EmojiService) Send(request *SendRequest) (*SendResponse, error) {

	rsp := &SendResponse{}
	return rsp, t.client.Call("emoji", "Send", request, rsp)

}

type FindRequest struct {
	// the alias code e.g :beer:
	Alias string `json:"alias"`
}

type FindResponse struct {
	// the unicode emoji 🍺
	Emoji string `json:"emoji"`
}

type FlagRequest struct {
	// country code e.g GB
	Code string `json:"code"`
}

type FlagResponse struct {
	// the emoji flag
	Flag string `json:"flag"`
}

type PrintRequest struct {
	// text including any alias e.g let's grab a :beer:
	Text string `json:"text"`
}

type PrintResponse struct {
	// text with rendered emojis
	Text string `json:"text"`
}

type SendRequest struct {
	// the name of the sender from e.g Alice
	From string `json:"from"`
	// message to send including emoji aliases
	Message string `json:"message"`
	// phone number to send to (including international dialing code)
	To string `json:"to"`
}

type SendResponse struct {
	// whether or not it succeeded
	Success bool `json:"success"`
}

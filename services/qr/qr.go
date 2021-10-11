package qr

import (
	"github.com/m3o/m3o-go/client"
)

func NewQrService(token string) *QrService {
	return &QrService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type QrService struct {
	client *client.Client
}

// Generate a QR code with a specific text and size
func (t *QrService) Generate(request *GenerateRequest) (*GenerateResponse, error) {
	rsp := &GenerateResponse{}
	return rsp, t.client.Call("qr", "Generate", request, rsp)
}

type GenerateRequest struct {
	// the size (height and width) in pixels of the generated QR code. Defaults to 256
	Size int64 `json:"size,string"`
	// the text to encode as a QR code (URL, phone number, email, etc)
	Text string `json:"text"`
}

type GenerateResponse struct {
	// link to the QR code image in PNG format
	Qr string `json:"qr"`
}

package otp

import (
	"go-micro.dev/v4/api/client"
)

type Otp interface {
	Generate(*GenerateRequest) (*GenerateResponse, error)
	Validate(*ValidateRequest) (*ValidateResponse, error)
}

func NewOtpService(token string) *OtpService {
	return &OtpService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type OtpService struct {
	client *client.Client
}

// Generate an OTP (one time pass) code
func (t *OtpService) Generate(request *GenerateRequest) (*GenerateResponse, error) {

	rsp := &GenerateResponse{}
	return rsp, t.client.Call("otp", "Generate", request, rsp)

}

// Validate the OTP code
func (t *OtpService) Validate(request *ValidateRequest) (*ValidateResponse, error) {

	rsp := &ValidateResponse{}
	return rsp, t.client.Call("otp", "Validate", request, rsp)

}

type GenerateRequest struct {
	// expiration in seconds (default: 60)
	Expiry int64 `json:"expiry,string"`
	// unique id, email or user to generate an OTP for
	Id string `json:"id"`
	// number of characters (default: 6)
	Size int64 `json:"size,string"`
}

type GenerateResponse struct {
	// one time pass code
	Code string `json:"code"`
}

type ValidateRequest struct {
	// one time pass code to validate
	Code string `json:"code"`
	// unique id, email or user for which the code was generated
	Id string `json:"id"`
}

type ValidateResponse struct {
	// returns true if the code is valid for the ID
	Success bool `json:"success"`
}

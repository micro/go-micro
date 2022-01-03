package id

import (
	"go-micro.dev/v4/api/client"
)

type Id interface {
	Generate(*GenerateRequest) (*GenerateResponse, error)
	Types(*TypesRequest) (*TypesResponse, error)
}

func NewIdService(token string) *IdService {
	return &IdService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type IdService struct {
	client *client.Client
}

// Generate a unique ID. Defaults to uuid.
func (t *IdService) Generate(request *GenerateRequest) (*GenerateResponse, error) {

	rsp := &GenerateResponse{}
	return rsp, t.client.Call("id", "Generate", request, rsp)

}

// List the types of IDs available. No query params needed.
func (t *IdService) Types(request *TypesRequest) (*TypesResponse, error) {

	rsp := &TypesResponse{}
	return rsp, t.client.Call("id", "Types", request, rsp)

}

type GenerateRequest struct {
	// type of id e.g uuid, shortid, snowflake (64 bit), bigflake (128 bit)
	Type string `json:"type"`
}

type GenerateResponse struct {
	// the unique id generated
	Id string `json:"id"`
	// the type of id generated
	Type string `json:"type"`
}

type TypesRequest struct {
}

type TypesResponse struct {
	Types []string `json:"types"`
}

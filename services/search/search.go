package search

import (
	"go-micro.dev/v4/api/client"
)

type Search interface {
	Vote(*VoteRequest) (*VoteResponse, error)
}

func NewSearchService(token string) *SearchService {
	return &SearchService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type SearchService struct {
	client *client.Client
}

// Vote to have the Search api launched faster!
func (t *SearchService) Vote(request *VoteRequest) (*VoteResponse, error) {

	rsp := &VoteResponse{}
	return rsp, t.client.Call("search", "Vote", request, rsp)

}

type VoteRequest struct {
	// optional message
	Message string `json:"message"`
}

type VoteResponse struct {
	// response message
	Message string `json:"message"`
}

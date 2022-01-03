package joke

import (
	"go-micro.dev/v4/api/client"
)

type Joke interface {
	Random(*RandomRequest) (*RandomResponse, error)
}

func NewJokeService(token string) *JokeService {
	return &JokeService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type JokeService struct {
	client *client.Client
}

// Get a random joke
func (t *JokeService) Random(request *RandomRequest) (*RandomResponse, error) {

	rsp := &RandomResponse{}
	return rsp, t.client.Call("joke", "Random", request, rsp)

}

type JokeInfo struct {
	Body     string `json:"body"`
	Category string `json:"category"`
	Id       string `json:"id"`
	Source   string `json:"source"`
	Title    string `json:"title"`
}

type RandomRequest struct {
	// the count of random jokes want, maximum: 10
	Count int32 `json:"count"`
}

type RandomResponse struct {
	Jokes []JokeInfo `json:"jokes"`
}

package avatar

import (
	"go-micro.dev/v4/api/client"
)

type Avatar interface {
	Generate(*GenerateRequest) (*GenerateResponse, error)
}

func NewAvatarService(token string) *AvatarService {
	return &AvatarService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type AvatarService struct {
	client *client.Client
}

//
func (t *AvatarService) Generate(request *GenerateRequest) (*GenerateResponse, error) {

	rsp := &GenerateResponse{}
	return rsp, t.client.Call("avatar", "Generate", request, rsp)

}

type GenerateRequest struct {
	// encode format of avatar image, `png` or `jpeg`, default is `jpeg`
	Format string `json:"format"`
	// avatar's gender, `male` or `female`, default is `male`
	Gender string `json:"gender"`
	// if upload to m3o CDN, default is `false`
	// if update = true, then it'll return the CDN url
	Upload bool `json:"upload"`
	// avatar's username, unique username will generates the unique avatar;
	// if username == "", will generate a random avatar in every request
	// if upload == true, username will be used as CDN filename rather than a random uuid string
	Username string `json:"username"`
}

type GenerateResponse struct {
	// base64encode string of the avatar image
	Base64 string `json:"base64"`
	// Micro's CDN url of the avatar image
	Url string `json:"url"`
}

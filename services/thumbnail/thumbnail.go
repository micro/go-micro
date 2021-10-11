package thumbnail

import (
	"github.com/m3o/m3o-go/client"
)

func NewThumbnailService(token string) *ThumbnailService {
	return &ThumbnailService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type ThumbnailService struct {
	client *client.Client
}

// Create a thumbnail screenshot by passing in a url, height and width
func (t *ThumbnailService) Screenshot(request *ScreenshotRequest) (*ScreenshotResponse, error) {
	rsp := &ScreenshotResponse{}
	return rsp, t.client.Call("thumbnail", "Screenshot", request, rsp)
}

type ScreenshotRequest struct {
	// height of the browser window, optional
	Height int32  `json:"height"`
	Url    string `json:"url"`
	// width of the browser window. optional
	Width int32 `json:"width"`
}

type ScreenshotResponse struct {
	ImageUrl string `json:"imageUrl"`
}

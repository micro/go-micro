package youtube

import (
	"go.m3o.com/client"
)

func NewYoutubeService(token string) *YoutubeService {
	return &YoutubeService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type YoutubeService struct {
	client *client.Client
}

// Search for videos on YouTube
func (t *YoutubeService) Search(request *SearchRequest) (*SearchResponse, error) {
	rsp := &SearchResponse{}
	return rsp, t.client.Call("youtube", "Search", request, rsp)
}

type SearchRequest struct {
	// Query to search for
	Query string `json:"query"`
}

type SearchResponse struct {
	// List of results for the query
	Results []SearchResult `json:"results"`
}

type SearchResult struct {
	// if live broadcast then indicates activity.
	// none, upcoming, live, completed
	Broadcasting string `json:"broadcasting"`
	// the channel id
	ChannelId string `json:"channelId"`
	// the channel title
	ChannelTitle string `json:"channelTitle"`
	// the result description
	Description string `json:"description"`
	// id of the result
	Id string `json:"id"`
	// kind of result; "video", "channel", "playlist"
	Kind string `json:"kind"`
	// published at time
	PublishedAt string `json:"publishedAt"`
	// title of the result
	Title string `json:"title"`
	// the associated url
	Url string `json:"url"`
}

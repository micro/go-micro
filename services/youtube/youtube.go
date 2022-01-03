package youtube

import (
	"go-micro.dev/v4/api/client"
)

type Youtube interface {
	Embed(*EmbedRequest) (*EmbedResponse, error)
	Search(*SearchRequest) (*SearchResponse, error)
}

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

// Embed a YouTube video
func (t *YoutubeService) Embed(request *EmbedRequest) (*EmbedResponse, error) {

	rsp := &EmbedResponse{}
	return rsp, t.client.Call("youtube", "Embed", request, rsp)

}

// Search for videos on YouTube
func (t *YoutubeService) Search(request *SearchRequest) (*SearchResponse, error) {

	rsp := &SearchResponse{}
	return rsp, t.client.Call("youtube", "Search", request, rsp)

}

type EmbedRequest struct {
	// provide the youtube url e.g https://www.youtube.com/watch?v=GWRWZu7XsJ0
	Url string `json:"url"`
}

type EmbedResponse struct {
	// the embeddable link e.g https://www.youtube.com/watch?v=GWRWZu7XsJ0
	EmbedUrl string `json:"embed_url"`
	// the script code
	HtmlScript string `json:"html_script"`
	// the full url
	LongUrl string `json:"long_url"`
	// the short url
	ShortUrl string `json:"short_url"`
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
	ChannelId string `json:"channel_id"`
	// the channel title
	ChannelTitle string `json:"channel_title"`
	// the result description
	Description string `json:"description"`
	// id of the result
	Id string `json:"id"`
	// kind of result; "video", "channel", "playlist"
	Kind string `json:"kind"`
	// published at time
	PublishedAt string `json:"published_at"`
	// title of the result
	Title string `json:"title"`
	// the associated url
	Url string `json:"url"`
}

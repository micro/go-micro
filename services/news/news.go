package news

import (
	"go-micro.dev/v4/api/client"
)

type News interface {
	Headlines(*HeadlinesRequest) (*HeadlinesResponse, error)
}

func NewNewsService(token string) *NewsService {
	return &NewsService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type NewsService struct {
	client *client.Client
}

// Get the latest news headlines
func (t *NewsService) Headlines(request *HeadlinesRequest) (*HeadlinesResponse, error) {

	rsp := &HeadlinesResponse{}
	return rsp, t.client.Call("news", "Headlines", request, rsp)

}

type Article struct {
	// categories
	Categories []string `json:"categories"`
	// article description
	Description string `json:"description"`
	// article id
	Id string `json:"id"`
	// image url
	ImageUrl string `json:"image_url"`
	// related keywords
	Keywords string `json:"keywords"`
	// the article language
	Language string `json:"language"`
	// the locale
	Locale string `json:"locale"`
	// time it was published
	PublishedAt string `json:"published_at"`
	// first 60 characters of article body
	Snippet string `json:"snippet"`
	// source of news
	Source string `json:"source"`
	// article title
	Title string `json:"title"`
	// url of the article
	Url string `json:"url"`
}

type HeadlinesRequest struct {
	// date published on in YYYY-MM-DD format
	Date string `json:"date"`
	// comma separated list of languages to retrieve in e.g en,es
	Language string `json:"language"`
	// comma separated list of countries to include e.g us,ca
	Locale string `json:"locale"`
}

type HeadlinesResponse struct {
	Articles []Article `json:"articles"`
}

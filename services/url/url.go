package url

import (
	"go-micro.dev/v4/api/client"
)

type Url interface {
	List(*ListRequest) (*ListResponse, error)
	Proxy(*ProxyRequest) (*ProxyResponse, error)
	Shorten(*ShortenRequest) (*ShortenResponse, error)
}

func NewUrlService(token string) *UrlService {
	return &UrlService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type UrlService struct {
	client *client.Client
}

// List all the shortened URLs
func (t *UrlService) List(request *ListRequest) (*ListResponse, error) {

	rsp := &ListResponse{}
	return rsp, t.client.Call("url", "List", request, rsp)

}

// Proxy returns the destination URL of a short URL.
func (t *UrlService) Proxy(request *ProxyRequest) (*ProxyResponse, error) {

	rsp := &ProxyResponse{}
	return rsp, t.client.Call("url", "Proxy", request, rsp)

}

// Shorten a long URL
func (t *UrlService) Shorten(request *ShortenRequest) (*ShortenResponse, error) {

	rsp := &ShortenResponse{}
	return rsp, t.client.Call("url", "Shorten", request, rsp)

}

type ListRequest struct {
	// filter by short URL, optional
	ShortUrl string `json:"shortURL"`
}

type ListResponse struct {
	UrlPairs *URLPair `json:"urlPairs"`
}

type ProxyRequest struct {
	// short url ID, without the domain, eg. if your short URL is
	// `m3o.one/u/someshorturlid` then pass in `someshorturlid`
	ShortUrl string `json:"shortURL"`
}

type ProxyResponse struct {
	DestinationUrl string `json:"destinationURL"`
}

type ShortenRequest struct {
	// the url to shorten
	DestinationUrl string `json:"destinationURL"`
}

type ShortenResponse struct {
	// the shortened url
	ShortUrl string `json:"shortURL"`
}

type URLPair struct {
	// time of creation
	Created string `json:"created"`
	// destination url
	DestinationUrl string `json:"destinationURL"`
	// shortened url
	ShortUrl string `json:"shortURL"`
}

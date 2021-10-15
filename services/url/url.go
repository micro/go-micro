package url

import (
	"github.com/m3o/m3o-go/client"
)

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

// List information on all the shortened URLs that you have created
func (t *UrlService) List(request *ListRequest) (*ListResponse, error) {
	rsp := &ListResponse{}
	return rsp, t.client.Call("url", "List", request, rsp)
}

// Proxy returns the destination URL of a short URL.
func (t *UrlService) Proxy(request *ProxyRequest) (*ProxyResponse, error) {
	rsp := &ProxyResponse{}
	return rsp, t.client.Call("url", "Proxy", request, rsp)
}

// Shortens a destination URL and returns a full short URL.
func (t *UrlService) Shorten(request *ShortenRequest) (*ShortenResponse, error) {
	rsp := &ShortenResponse{}
	return rsp, t.client.Call("url", "Shorten", request, rsp)
}

type ListRequest struct {
	// filter by short URL, optional
	ShortUrl string `json:"shortUrl"`
}

type ListResponse struct {
	UrlPairs *URLPair `json:"urlPairs"`
}

type ProxyRequest struct {
	// short url ID, without the domain, eg. if your short URL is
	// `m3o.one/u/someshorturlid` then pass in `someshorturlid`
	ShortUrl string `json:"shortUrl"`
}

type ProxyResponse struct {
	DestinationUrl string `json:"destinationUrl"`
}

type ShortenRequest struct {
	DestinationUrl string `json:"destinationUrl"`
}

type ShortenResponse struct {
	ShortUrl string `json:"shortUrl"`
}

type URLPair struct {
	Created        int64  `json:"created,string"`
	DestinationUrl string `json:"destinationUrl"`
	// HitCount keeps track many times the short URL has been resolved.
	// Hitcount only gets saved to disk (database) after every 10th hit, so
	// its not intended to be 100% accurate, more like an almost correct estimate.
	HitCount int64  `json:"hitCount,string"`
	Owner    string `json:"owner"`
	ShortUrl string `json:"shortUrl"`
}

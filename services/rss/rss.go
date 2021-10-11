package rss

import (
	"github.com/m3o/m3o-go/client"
)

func NewRssService(token string) *RssService {
	return &RssService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type RssService struct {
	client *client.Client
}

// Add a new RSS feed with a name, url, and category
func (t *RssService) Add(request *AddRequest) (*AddResponse, error) {
	rsp := &AddResponse{}
	return rsp, t.client.Call("rss", "Add", request, rsp)
}

// Get an RSS feed by name. If no name is given, all feeds are returned. Default limit is 25 entries.
func (t *RssService) Feed(request *FeedRequest) (*FeedResponse, error) {
	rsp := &FeedResponse{}
	return rsp, t.client.Call("rss", "Feed", request, rsp)
}

// List the saved RSS fields
func (t *RssService) List(request *ListRequest) (*ListResponse, error) {
	rsp := &ListResponse{}
	return rsp, t.client.Call("rss", "List", request, rsp)
}

// Remove an RSS feed by name
func (t *RssService) Remove(request *RemoveRequest) (*RemoveResponse, error) {
	rsp := &RemoveResponse{}
	return rsp, t.client.Call("rss", "Remove", request, rsp)
}

type AddRequest struct {
	// category to add e.g news
	Category string `json:"category"`
	// rss feed name
	// eg. a16z
	Name string `json:"name"`
	// rss feed url
	// eg. http://a16z.com/feed/
	Url string `json:"url"`
}

type AddResponse struct {
}

type Entry struct {
	// article content
	Content string `json:"content"`
	// data of the entry
	Date string `json:"date"`
	// the rss feed where it came from
	Feed string `json:"feed"`
	// unique id of the entry
	Id string `json:"id"`
	// rss feed url of the entry
	Link string `json:"link"`
	// article summary
	Summary string `json:"summary"`
	// title of the entry
	Title string `json:"title"`
}

type Feed struct {
	// category of the feed e.g news
	Category string `json:"category"`
	// unique id
	Id string `json:"id"`
	// rss feed name
	// eg. a16z
	Name string `json:"name"`
	// rss feed url
	// eg. http://a16z.com/feed/
	Url string `json:"url"`
}

type FeedRequest struct {
	// limit entries returned
	Limit int64 `json:"limit,string"`
	// rss feed name
	Name string `json:"name"`
	// offset entries
	Offset int64 `json:"offset,string"`
}

type FeedResponse struct {
	Entries []Entry `json:"entries"`
}

type ListRequest struct {
}

type ListResponse struct {
	Feeds []Feed `json:"feeds"`
}

type RemoveRequest struct {
	// rss feed name
	// eg. a16z
	Name string `json:"name"`
}

type RemoveResponse struct {
}

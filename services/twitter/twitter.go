package twitter

import (
	"github.com/m3o/m3o-go/client"
)

func NewTwitterService(token string) *TwitterService {
	return &TwitterService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type TwitterService struct {
	client *client.Client
}

// Search for tweets with a simple query
func (t *TwitterService) Search(request *SearchRequest) (*SearchResponse, error) {
	rsp := &SearchResponse{}
	return rsp, t.client.Call("twitter", "Search", request, rsp)
}

// Get the timeline for a given user
func (t *TwitterService) Timeline(request *TimelineRequest) (*TimelineResponse, error) {
	rsp := &TimelineResponse{}
	return rsp, t.client.Call("twitter", "Timeline", request, rsp)
}

// Get the current global trending topics
func (t *TwitterService) Trends(request *TrendsRequest) (*TrendsResponse, error) {
	rsp := &TrendsResponse{}
	return rsp, t.client.Call("twitter", "Trends", request, rsp)
}

// Get a user's twitter profile
func (t *TwitterService) User(request *UserRequest) (*UserResponse, error) {
	rsp := &UserResponse{}
	return rsp, t.client.Call("twitter", "User", request, rsp)
}

type Profile struct {
	// the account creation date
	CreatedAt string `json:"createdAt"`
	// the user description
	Description string `json:"description"`
	// the follower count
	Followers int64 `json:"followers,string"`
	// the user id
	Id int64 `json:"id,string"`
	// The user's profile picture
	ImageUrl string `json:"imageUrl"`
	// the user's location
	Location string `json:"location"`
	// display name of the user
	Name string `json:"name"`
	// if the account is private
	Private bool `json:"private"`
	// the username
	Username string `json:"username"`
	// if the account is verified
	Verified bool `json:"verified"`
}

type SearchRequest struct {
	// number of tweets to return. default: 20
	Limit int32 `json:"limit"`
	// the query to search for
	Query string `json:"query"`
}

type SearchResponse struct {
	// the related tweets for the search
	Tweets []Tweet `json:"tweets"`
}

type TimelineRequest struct {
	// number of tweets to return. default: 20
	Limit int32 `json:"limit"`
	// the username to request the timeline for
	Username string `json:"username"`
}

type TimelineResponse struct {
	// The recent tweets for the user
	Tweets []Tweet `json:"tweets"`
}

type Trend struct {
	// name of the trend
	Name string `json:"name"`
	// the volume of tweets in last 24 hours
	TweetVolume int64 `json:"tweetVolume,string"`
	// the twitter url
	Url string `json:"url"`
}

type TrendsRequest struct {
}

type TrendsResponse struct {
	// a list of trending topics
	Trends []Trend `json:"trends"`
}

type Tweet struct {
	// time of tweet
	CreatedAt string `json:"createdAt"`
	// number of times favourited
	FavouritedCount int64 `json:"favouritedCount,string"`
	// id of the tweet
	Id int64 `json:"id,string"`
	// number of times retweeted
	RetweetedCount int64 `json:"retweetedCount,string"`
	// text of the tweet
	Text string `json:"text"`
	// username of the person who tweeted
	Username string `json:"username"`
}

type UserRequest struct {
	// the username to lookup
	Username string `json:"username"`
}

type UserResponse struct {
	// The requested user profile
	Profile *Profile `json:"profile"`
	// the current user status
	Status *Tweet `json:"status"`
}

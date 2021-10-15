package gifs

import (
	"github.com/m3o/m3o-go/client"
)

func NewGifsService(token string) *GifsService {
	return &GifsService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type GifsService struct {
	client *client.Client
}

// Search for a gif
func (t *GifsService) Search(request *SearchRequest) (*SearchResponse, error) {
	rsp := &SearchResponse{}
	return rsp, t.client.Call("gifs", "Search", request, rsp)
}

type Gif struct {
	// URL used for embedding the GIF
	EmbedUrl string `json:"embedUrl"`
	// The ID of the GIF
	Id string `json:"id"`
	// The different formats available for this GIF
	Images *ImageFormats `json:"images"`
	// The content rating for the GIF
	Rating string `json:"rating"`
	// A short URL for this GIF
	ShortUrl string `json:"shortUrl"`
	// The slug used in the GIF's URL
	Slug string `json:"slug"`
	// The page on which this GIF was found
	Source string `json:"source"`
	// The title for this GIF
	Title string `json:"title"`
	// The URL for this GIF
	Url string `json:"url"`
}

type ImageFormat struct {
	// height
	Height int32 `json:"height"`
	// size of the MP4 version
	Mp4size int32 `json:"mp4size"`
	// URL to an MP4 version of the gif
	Mp4url string `json:"mp4url"`
	// size in bytes
	Size int32 `json:"size"`
	// URL of the gif
	Url string `json:"url"`
	// size of the webp version
	WebpSize int32 `json:"webpSize"`
	// URL to a webp version of the gif
	WebpUrl string `json:"webpUrl"`
	// width
	Width int32 `json:"width"`
}

type ImageFormats struct {
	// A downsized version of the GIF < 2MB
	Downsized *ImageFormat `json:"downsized"`
	// A downsized version of the GIF < 8MB
	DownsizedLarge *ImageFormat `json:"downsizedLarge"`
	// A downsized version of the GIF < 5MB
	DownsizedMedium *ImageFormat `json:"downsizedMedium"`
	// A downsized version of the GIF < 200kb
	DownsizedSmall *ImageFormat `json:"downsizedSmall"`
	// Static image of the downsized version of the GIF
	DownsizedStill *ImageFormat `json:"downsizedStill"`
	// Version of the GIF with fixed height of 200 pixels. Good for mobile use
	FixedHeight *ImageFormat `json:"fixedHeight"`
	// Version of the GIF with fixed height of 200 pixels and number of frames reduced to 6
	FixedHeightDownsampled *ImageFormat `json:"fixedHeightDownsampled"`
	// Version of the GIF with fixed height of 100 pixels. Good for mobile keyboards
	FixedHeightSmall *ImageFormat `json:"fixedHeightSmall"`
	// Static image of the GIF with fixed height of 100 pixels
	FixedHeightSmallStill *ImageFormat `json:"fixedHeightSmallStill"`
	// Static image of the GIF with fixed height of 200 pixels
	FixedHeightStill *ImageFormat `json:"fixedHeightStill"`
	// Version of the GIF with fixed width of 200 pixels. Good for mobile use
	FixedWidth *ImageFormat `json:"fixedWidth"`
	// Version of the GIF with fixed width of 200 pixels and number of frames reduced to 6
	FixedWidthDownsampled *ImageFormat `json:"fixedWidthDownsampled"`
	// Version of the GIF with fixed width of 100 pixels. Good for mobile keyboards
	FixedWidthSmall *ImageFormat `json:"fixedWidthSmall"`
	// Static image of the GIF with fixed width of 100 pixels
	FixedWidthSmallStill *ImageFormat `json:"fixedWidthSmallStill"`
	// Static image of the GIF with fixed width of 200 pixels
	FixedWidthStill *ImageFormat `json:"fixedWidthStill"`
	// 15 second version of the GIF looping
	Looping *ImageFormat `json:"looping"`
	// The original GIF. Good for desktop use
	Original *ImageFormat `json:"original"`
	// Static image of the original version of the GIF
	OriginalStill *ImageFormat `json:"originalStill"`
	// mp4 version of the GIF <50kb displaying first 1-2 secs
	Preview *ImageFormat `json:"preview"`
	// Version of the GIF <50kb displaying first 1-2 secs
	PreviewGif *ImageFormat `json:"previewGif"`
}

type Pagination struct {
	// total number returned in this response
	Count int32 `json:"count"`
	// position in pagination
	Offset int32 `json:"offset"`
	// total number of results available
	TotalCount int32 `json:"totalCount"`
}

type SearchRequest struct {
	// ISO 2 letter language code for regional content
	Lang string `json:"lang"`
	// Max number of gifs to return. Defaults to 25
	Limit int32 `json:"limit"`
	// The start position of results (used with pagination)
	Offset int32 `json:"offset"`
	// The search term
	Query string `json:"query"`
	// Apply age related content filter. "g", "pg", "pg-13", or "r". Defaults to "g"
	Rating string `json:"rating"`
}

type SearchResponse struct {
	// list of results
	Data []Gif `json:"data"`
	// information on pagination
	Pagination *Pagination `json:"pagination"`
}

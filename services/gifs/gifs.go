package gifs

import (
	"go-micro.dev/v4/api/client"
)

type Gifs interface {
	Search(*SearchRequest) (*SearchResponse, error)
}

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

// Search for a GIF
func (t *GifsService) Search(request *SearchRequest) (*SearchResponse, error) {

	rsp := &SearchResponse{}
	return rsp, t.client.Call("gifs", "Search", request, rsp)

}

type Gif struct {
	// URL used for embedding the GIF
	EmbedUrl string `json:"embed_url"`
	// The ID of the GIF
	Id string `json:"id"`
	// The different formats available for this GIF
	Images *ImageFormats `json:"images"`
	// The content rating for the GIF
	Rating string `json:"rating"`
	// A short URL for this GIF
	ShortUrl string `json:"short_url"`
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
	Mp4Size int32 `json:"mp4_size"`
	// URL to an MP4 version of the gif
	Mp4Url string `json:"mp4_url"`
	// size in bytes
	Size int32 `json:"size"`
	// URL of the gif
	Url string `json:"url"`
	// size of the webp version
	WebpSize int32 `json:"webp_size"`
	// URL to a webp version of the gif
	WebpUrl string `json:"webp_url"`
	// width
	Width int32 `json:"width"`
}

type ImageFormats struct {
	// A downsized version of the GIF < 2MB
	Downsized *ImageFormat `json:"downsized"`
	// A downsized version of the GIF < 8MB
	DownsizedLarge *ImageFormat `json:"downsized_large"`
	// A downsized version of the GIF < 5MB
	DownsizedMedium *ImageFormat `json:"downsized_medium"`
	// A downsized version of the GIF < 200kb
	DownsizedSmall *ImageFormat `json:"downsized_small"`
	// Static image of the downsized version of the GIF
	DownsizedStill *ImageFormat `json:"downsized_still"`
	// Version of the GIF with fixed height of 200 pixels. Good for mobile use
	FixedHeight *ImageFormat `json:"fixed_height"`
	// Version of the GIF with fixed height of 200 pixels and number of frames reduced to 6
	FixedHeightDownsampled *ImageFormat `json:"fixed_height_downsampled"`
	// Version of the GIF with fixed height of 100 pixels. Good for mobile keyboards
	FixedHeightSmall *ImageFormat `json:"fixed_height_small"`
	// Static image of the GIF with fixed height of 100 pixels
	FixedHeightSmallStill *ImageFormat `json:"fixed_height_small_still"`
	// Static image of the GIF with fixed height of 200 pixels
	FixedHeightStill *ImageFormat `json:"fixed_height_still"`
	// Version of the GIF with fixed width of 200 pixels. Good for mobile use
	FixedWidth *ImageFormat `json:"fixed_width"`
	// Version of the GIF with fixed width of 200 pixels and number of frames reduced to 6
	FixedWidthDownsampled *ImageFormat `json:"fixed_width_downsampled"`
	// Version of the GIF with fixed width of 100 pixels. Good for mobile keyboards
	FixedWidthSmall *ImageFormat `json:"fixed_width_small"`
	// Static image of the GIF with fixed width of 100 pixels
	FixedWidthSmallStill *ImageFormat `json:"fixed_width_small_still"`
	// Static image of the GIF with fixed width of 200 pixels
	FixedWidthStill *ImageFormat `json:"fixed_width_still"`
	// 15 second version of the GIF looping
	Looping *ImageFormat `json:"looping"`
	// The original GIF. Good for desktop use
	Original *ImageFormat `json:"original"`
	// Static image of the original version of the GIF
	OriginalStill *ImageFormat `json:"original_still"`
	// mp4 version of the GIF <50kb displaying first 1-2 secs
	Preview *ImageFormat `json:"preview"`
	// Version of the GIF <50kb displaying first 1-2 secs
	PreviewGif *ImageFormat `json:"preview_gif"`
}

type Pagination struct {
	// total number returned in this response
	Count int32 `json:"count"`
	// position in pagination
	Offset int32 `json:"offset"`
	// total number of results available
	TotalCount int32 `json:"total_count"`
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

package movie

import (
	"go-micro.dev/v4/api/client"
)

type Movie interface {
	Search(*SearchRequest) (*SearchResponse, error)
}

func NewMovieService(token string) *MovieService {
	return &MovieService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type MovieService struct {
	client *client.Client
}

// Search for movies by simple text search
func (t *MovieService) Search(request *SearchRequest) (*SearchResponse, error) {

	rsp := &SearchResponse{}
	return rsp, t.client.Call("movie", "Search", request, rsp)

}

type MovieInfo struct {
	Adult            bool    `json:"adult"`
	BackdropPath     string  `json:"backdrop_path"`
	GenreIds         int32   `json:"genre_ids"`
	Id               int32   `json:"id"`
	OriginalLanguage string  `json:"original_language"`
	OriginalTitle    string  `json:"original_title"`
	Overview         string  `json:"overview"`
	Popularity       float64 `json:"popularity"`
	PosterPath       string  `json:"poster_path"`
	ReleaseDate      string  `json:"release_date"`
	Title            string  `json:"title"`
	Video            bool    `json:"video"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int32   `json:"vote_count"`
}

type SearchRequest struct {
	// a ISO 639-1 value to display translated data
	Language string `json:"language"`
	// page to query
	Page int32 `json:"page"`
	// year of release
	PrimaryReleaseYear int32 `json:"primary_release_year"`
	// a text query to search
	Query string `json:"query"`
	// a ISO 3166-1 code to filter release dates.
	Region string `json:"region"`
	// year of making
	Year int32 `json:"year"`
}

type SearchResponse struct {
	Page         int32       `json:"page"`
	Results      []MovieInfo `json:"results"`
	TotalPages   int32       `json:"total_pages"`
	TotalResults int32       `json:"total_results"`
}

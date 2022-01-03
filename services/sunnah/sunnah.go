package sunnah

import (
	"go-micro.dev/v4/api/client"
)

type Sunnah interface {
	Books(*BooksRequest) (*BooksResponse, error)
	Chapters(*ChaptersRequest) (*ChaptersResponse, error)
	Collections(*CollectionsRequest) (*CollectionsResponse, error)
	Hadiths(*HadithsRequest) (*HadithsResponse, error)
}

func NewSunnahService(token string) *SunnahService {
	return &SunnahService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type SunnahService struct {
	client *client.Client
}

// Get a list of books from within a collection. A book can contain many chapters
// each with its own hadiths.
func (t *SunnahService) Books(request *BooksRequest) (*BooksResponse, error) {

	rsp := &BooksResponse{}
	return rsp, t.client.Call("sunnah", "Books", request, rsp)

}

// Get all the chapters of a given book within a collection.
func (t *SunnahService) Chapters(request *ChaptersRequest) (*ChaptersResponse, error) {

	rsp := &ChaptersResponse{}
	return rsp, t.client.Call("sunnah", "Chapters", request, rsp)

}

// Get a list of available collections. A collection is
// a compilation of hadiths collected and written by an author.
func (t *SunnahService) Collections(request *CollectionsRequest) (*CollectionsResponse, error) {

	rsp := &CollectionsResponse{}
	return rsp, t.client.Call("sunnah", "Collections", request, rsp)

}

// Hadiths returns a list of hadiths and their corresponding text for a
// given book within a collection.
func (t *SunnahService) Hadiths(request *HadithsRequest) (*HadithsResponse, error) {

	rsp := &HadithsResponse{}
	return rsp, t.client.Call("sunnah", "Hadiths", request, rsp)

}

type Book struct {
	// arabic name of the book
	ArabicName string `json:"arabic_name"`
	// number of hadiths in the book
	Hadiths int32 `json:"hadiths"`
	// number of the book e.g 1
	Id int32 `json:"id"`
	// name of the book
	Name string `json:"name"`
}

type BooksRequest struct {
	// Name of the collection
	Collection string `json:"collection"`
	// Limit the number of books returned
	Limit int32 `json:"limit"`
	// The page in the pagination
	Page int32 `json:"page"`
}

type BooksResponse struct {
	// A list of books
	Books []Book `json:"books"`
	// Name of the collection
	Collection string `json:"collection"`
	// The limit specified
	Limit int32 `json:"limit"`
	// The page requested
	Page int32 `json:"page"`
	// The total overall books
	Total int32 `json:"total"`
}

type Chapter struct {
	// arabic title
	ArabicTitle string `json:"arabic_title"`
	// the book number
	Book int32 `json:"book"`
	// the chapter id e.g 1
	Id int32 `json:"id"`
	// the chapter key e.g 1.00
	Key string `json:"key"`
	// title of the chapter
	Title string `json:"title"`
}

type ChaptersRequest struct {
	// number of the book
	Book int32 `json:"book"`
	// name of the collection
	Collection string `json:"collection"`
	// Limit the number of chapters returned
	Limit int32 `json:"limit"`
	// The page in the pagination
	Page int32 `json:"page"`
}

type ChaptersResponse struct {
	// number of the book
	Book int32 `json:"book"`
	// The chapters of the book
	Chapters []Chapter `json:"chapters"`
	// name of the collection
	Collection string `json:"collection"`
	// Limit the number of chapters returned
	Limit int32 `json:"limit"`
	// The page in the pagination
	Page int32 `json:"page"`
	// Total chapters in the book
	Total int32 `json:"total"`
}

type Collection struct {
	// Arabic title if available
	ArabicTitle string `json:"arabic_title"`
	// Total hadiths in the collection
	Hadiths int32 `json:"hadiths"`
	// Name of the collection e.g bukhari
	Name string `json:"name"`
	// An introduction explaining the collection
	Summary string `json:"summary"`
	// Title of the collection e.g Sahih al-Bukhari
	Title string `json:"title"`
}

type CollectionsRequest struct {
	// Number of collections to limit to
	Limit int32 `json:"limit"`
	// The page in the pagination
	Page int32 `json:"page"`
}

type CollectionsResponse struct {
	Collections []Collection `json:"collections"`
}

type Hadith struct {
	// the arabic chapter title
	ArabicChapterTitle string `json:"arabic_chapter_title"`
	// the arabic text
	ArabicText string `json:"arabic_text"`
	// the chapter id
	Chapter int32 `json:"chapter"`
	// the chapter key
	ChapterKey string `json:"chapter_key"`
	// the chapter title
	ChapterTitle string `json:"chapter_title"`
	// hadith id
	Id int32 `json:"id"`
	// hadith text
	Text string `json:"text"`
}

type HadithsRequest struct {
	// number of the book
	Book int32 `json:"book"`
	// name of the collection
	Collection string `json:"collection"`
	// Limit the number of hadiths
	Limit int32 `json:"limit"`
	// The page in the pagination
	Page int32 `json:"page"`
}

type HadithsResponse struct {
	// number of the book
	Book int32 `json:"book"`
	// name of the collection
	Collection string `json:"collection"`
	// The hadiths of the book
	Hadiths []Hadith `json:"hadiths"`
	// Limit the number of hadiths returned
	Limit int32 `json:"limit"`
	// The page in the pagination
	Page int32 `json:"page"`
	// Total hadiths in the  book
	Total int32 `json:"total"`
}

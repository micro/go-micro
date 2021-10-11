package quran

import (
	"github.com/m3o/m3o-go/client"
)

func NewQuranService(token string) *QuranService {
	return &QuranService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type QuranService struct {
	client *client.Client
}

// List the Chapters (surahs) of the Quran
func (t *QuranService) Chapters(request *ChaptersRequest) (*ChaptersResponse, error) {
	rsp := &ChaptersResponse{}
	return rsp, t.client.Call("quran", "Chapters", request, rsp)
}

// Search the Quran for any form of query or questions
func (t *QuranService) Search(request *SearchRequest) (*SearchResponse, error) {
	rsp := &SearchResponse{}
	return rsp, t.client.Call("quran", "Search", request, rsp)
}

// Get a summary for a given chapter (surah)
func (t *QuranService) Summary(request *SummaryRequest) (*SummaryResponse, error) {
	rsp := &SummaryResponse{}
	return rsp, t.client.Call("quran", "Summary", request, rsp)
}

// Lookup the verses (ayahs) for a chapter including
// translation, interpretation and breakdown by individual
// words.
func (t *QuranService) Verses(request *VersesRequest) (*VersesResponse, error) {
	rsp := &VersesResponse{}
	return rsp, t.client.Call("quran", "Verses", request, rsp)
}

type Chapter struct {
	// The arabic name of the chapter
	ArabicName string `json:"arabicName"`
	// The complex name of the chapter
	ComplexName string `json:"complexName"`
	// The id of the chapter as a number e.g 1
	Id int32 `json:"id"`
	// The simple name of the chapter
	Name string `json:"name"`
	// The pages from and to e.g 1, 1
	Pages []int32 `json:"pages"`
	// Should the chapter start with bismillah
	PrefixBismillah bool `json:"prefixBismillah"`
	// The order in which it was revealed
	RevelationOrder int32 `json:"revelationOrder"`
	// The place of revelation
	RevelationPlace string `json:"revelationPlace"`
	// The translated name
	TranslatedName string `json:"translatedName"`
	// The number of verses in the chapter
	Verses int32 `json:"verses"`
}

type ChaptersRequest struct {
	// Specify the language e.g en
	Language string `json:"language"`
}

type ChaptersResponse struct {
	Chapters []Chapter `json:"chapters"`
}

type Interpretation struct {
	// The unique id of the interpretation
	Id int32 `json:"id"`
	// The source of the interpretation
	Source string `json:"source"`
	// The translated text
	Text string `json:"text"`
}

type Result struct {
	// The associated arabic text
	Text string `json:"text"`
	// The related translations to the text
	Translations []Translation `json:"translations"`
	// The unique verse id across the Quran
	VerseId int32 `json:"verseId"`
	// The verse key e.g 1:1
	VerseKey string `json:"verseKey"`
}

type SearchRequest struct {
	// The language for translation
	Language string `json:"language"`
	// The number of results to return
	Limit int32 `json:"limit"`
	// The pagination number
	Page int32 `json:"page"`
	// The query to ask
	Query string `json:"query"`
}

type SearchResponse struct {
	// The current page
	Page int32 `json:"page"`
	// The question asked
	Query string `json:"query"`
	// The results for the query
	Results []Result `json:"results"`
	// The total pages
	TotalPages int32 `json:"totalPages"`
	// The total results returned
	TotalResults int32 `json:"totalResults"`
}

type SummaryRequest struct {
	// The chapter id e.g 1
	Chapter int32 `json:"chapter"`
	// Specify the language e.g en
	Language string `json:"language"`
}

type SummaryResponse struct {
	// The chapter id
	Chapter int32 `json:"chapter"`
	// The source of the summary
	Source string `json:"source"`
	// The short summary for the chapter
	Summary string `json:"summary"`
	// The full description for the chapter
	Text string `json:"text"`
}

type Translation struct {
	// The unique id of the translation
	Id int32 `json:"id"`
	// The source of the translation
	Source string `json:"source"`
	// The translated text
	Text string `json:"text"`
}

type Verse struct {
	// The unique id of the verse in the whole book
	Id int32 `json:"id"`
	// The interpretations of the verse
	Interpretations []Interpretation `json:"interpretations"`
	// The key of this verse (chapter:verse) e.g 1:1
	Key string `json:"key"`
	// The verse number in this chapter
	Number int32 `json:"number"`
	// The page of the Quran this verse is on
	Page int32 `json:"page"`
	// The arabic text for this verse
	Text string `json:"text"`
	// The basic translation of the verse
	TranslatedText string `json:"translatedText"`
	// The alternative translations for the verse
	Translations []Translation `json:"translations"`
	// The phonetic transliteration from arabic
	Transliteration string `json:"transliteration"`
	// The individual words within the verse (Ayah)
	Words []Word `json:"words"`
}

type VersesRequest struct {
	// The chapter id to retrieve
	Chapter int32 `json:"chapter"`
	// Return the interpretation (tafsir)
	Interpret bool `json:"interpret"`
	// The language of translation
	Language string `json:"language"`
	// The verses per page
	Limit int32 `json:"limit"`
	// The page number to request
	Page int32 `json:"page"`
	// Return alternate translations
	Translate bool `json:"translate"`
	// Return the individual words with the verses
	Words bool `json:"words"`
}

type VersesResponse struct {
	// The chapter requested
	Chapter int32 `json:"chapter"`
	// The page requested
	Page int32 `json:"page"`
	// The total pages
	TotalPages int32 `json:"totalPages"`
	// The verses on the page
	Verses []Verse `json:"verses"`
}

type Word struct {
	// The character type e.g word, end
	CharType string `json:"charType"`
	// The QCF v2 font code
	Code string `json:"code"`
	// The id of the word within the verse
	Id int32 `json:"id"`
	// The line number
	Line int32 `json:"line"`
	// The page number
	Page int32 `json:"page"`
	// The position of the word
	Position int32 `json:"position"`
	// The arabic text for this word
	Text string `json:"text"`
	// The translated text
	Translation string `json:"translation"`
	// The transliteration text
	Transliteration string `json:"transliteration"`
}

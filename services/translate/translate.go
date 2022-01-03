package translate

import (
	"go-micro.dev/v4/api/client"
)

type Translate interface {
	Text(*TextRequest) (*TextResponse, error)
}

func NewTranslateService(token string) *TranslateService {
	return &TranslateService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type TranslateService struct {
	client *client.Client
}

// Basic text translation
func (t *TranslateService) Text(request *TextRequest) (*TextResponse, error) {

	rsp := &TextResponse{}
	return rsp, t.client.Call("translate", "Text", request, rsp)

}

type TextRequest struct {
	// The contents to be translated
	Content string `json:"content"`
	// The string format, `text` or `html`
	Format string `json:"format"`
	// The model to use for translation, `nmt` or `base`,
	// See https://cloud.google.com/translate/docs/advanced/translating-text-v3#comparing-models for more information
	Model string `json:"model"`
	// Source language, format in ISO-639-1 codes
	// See https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes for more information
	Source string `json:"source"`
	// Target language, format in ISO-639-1 codes
	// See https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes for more information
	Target string `json:"target"`
}

type TextResponse struct {
	// The translated text
	Translation *Translation `json:"translation"`
}

type Translation struct {
	// The model used in translation
	Model string `json:"model"`
	// The source of the query string
	Source string `json:"source"`
	// The translation result
	Text string `json:"text"`
}

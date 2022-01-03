package crypto

import (
	"go-micro.dev/v4/api/client"
)

type Crypto interface {
	History(*HistoryRequest) (*HistoryResponse, error)
	News(*NewsRequest) (*NewsResponse, error)
	Price(*PriceRequest) (*PriceResponse, error)
	Quote(*QuoteRequest) (*QuoteResponse, error)
}

func NewCryptoService(token string) *CryptoService {
	return &CryptoService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type CryptoService struct {
	client *client.Client
}

// Returns the history for the previous close
func (t *CryptoService) History(request *HistoryRequest) (*HistoryResponse, error) {

	rsp := &HistoryResponse{}
	return rsp, t.client.Call("crypto", "History", request, rsp)

}

// Get news related to a currency
func (t *CryptoService) News(request *NewsRequest) (*NewsResponse, error) {

	rsp := &NewsResponse{}
	return rsp, t.client.Call("crypto", "News", request, rsp)

}

// Get the last price for a given crypto ticker
func (t *CryptoService) Price(request *PriceRequest) (*PriceResponse, error) {

	rsp := &PriceResponse{}
	return rsp, t.client.Call("crypto", "Price", request, rsp)

}

// Get the last quote for a given crypto ticker
func (t *CryptoService) Quote(request *QuoteRequest) (*QuoteResponse, error) {

	rsp := &QuoteResponse{}
	return rsp, t.client.Call("crypto", "Quote", request, rsp)

}

type Article struct {
	// the date published
	Date string `json:"date"`
	// its description
	Description string `json:"description"`
	// the source
	Source string `json:"source"`
	// title of the article
	Title string `json:"title"`
	// the source url
	Url string `json:"url"`
}

type HistoryRequest struct {
	// the crypto symbol e.g BTCUSD
	Symbol string `json:"symbol"`
}

type HistoryResponse struct {
	// the close price
	Close float64 `json:"close"`
	// the date
	Date string `json:"date"`
	// the peak price
	High float64 `json:"high"`
	// the low price
	Low float64 `json:"low"`
	// the open price
	Open float64 `json:"open"`
	// the crypto symbol
	Symbol string `json:"symbol"`
	// the volume
	Volume float64 `json:"volume"`
}

type NewsRequest struct {
	// cryptocurrency ticker to request news for e.g BTC
	Symbol string `json:"symbol"`
}

type NewsResponse struct {
	// list of articles
	Articles []Article `json:"articles"`
	// symbol requested for
	Symbol string `json:"symbol"`
}

type PriceRequest struct {
	// the crypto symbol e.g BTCUSD
	Symbol string `json:"symbol"`
}

type PriceResponse struct {
	// the last price
	Price float64 `json:"price"`
	// the crypto symbol e.g BTCUSD
	Symbol string `json:"symbol"`
}

type QuoteRequest struct {
	// the crypto symbol e.g BTCUSD
	Symbol string `json:"symbol"`
}

type QuoteResponse struct {
	// the asking price
	AskPrice float64 `json:"ask_price"`
	// the ask size
	AskSize float64 `json:"ask_size"`
	// the bidding price
	BidPrice float64 `json:"bid_price"`
	// the bid size
	BidSize float64 `json:"bid_size"`
	// the crypto symbol
	Symbol string `json:"symbol"`
	// the UTC timestamp of the quote
	Timestamp string `json:"timestamp"`
}

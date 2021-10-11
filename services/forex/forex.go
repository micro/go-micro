package forex

import (
	"github.com/m3o/m3o-go/client"
)

func NewForexService(token string) *ForexService {
	return &ForexService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type ForexService struct {
	client *client.Client
}

// Returns the data for the previous close
func (t *ForexService) History(request *HistoryRequest) (*HistoryResponse, error) {
	rsp := &HistoryResponse{}
	return rsp, t.client.Call("forex", "History", request, rsp)
}

// Get the latest price for a given forex ticker
func (t *ForexService) Price(request *PriceRequest) (*PriceResponse, error) {
	rsp := &PriceResponse{}
	return rsp, t.client.Call("forex", "Price", request, rsp)
}

// Get the latest quote for the forex
func (t *ForexService) Quote(request *QuoteRequest) (*QuoteResponse, error) {
	rsp := &QuoteResponse{}
	return rsp, t.client.Call("forex", "Quote", request, rsp)
}

type HistoryRequest struct {
	// the forex symbol e.g GBPUSD
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
	// the forex symbol
	Symbol string `json:"symbol"`
	// the volume
	Volume float64 `json:"volume"`
}

type PriceRequest struct {
	// forex symbol e.g GBPUSD
	Symbol string `json:"symbol"`
}

type PriceResponse struct {
	// the last price
	Price float64 `json:"price"`
	// the forex symbol e.g GBPUSD
	Symbol string `json:"symbol"`
}

type QuoteRequest struct {
	// the forex symbol e.g GBPUSD
	Symbol string `json:"symbol"`
}

type QuoteResponse struct {
	// the asking price
	AskPrice float64 `json:"askPrice"`
	// the bidding price
	BidPrice float64 `json:"bidPrice"`
	// the forex symbol
	Symbol string `json:"symbol"`
	// the UTC timestamp of the quote
	Timestamp string `json:"timestamp"`
}

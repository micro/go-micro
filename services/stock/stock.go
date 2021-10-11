package stock

import (
	"github.com/m3o/m3o-go/client"
)

func NewStockService(token string) *StockService {
	return &StockService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type StockService struct {
	client *client.Client
}

// Get the historic open-close for a given day
func (t *StockService) History(request *HistoryRequest) (*HistoryResponse, error) {
	rsp := &HistoryResponse{}
	return rsp, t.client.Call("stock", "History", request, rsp)
}

// Get the historic order book and each trade by timestamp
func (t *StockService) OrderBook(request *OrderBookRequest) (*OrderBookResponse, error) {
	rsp := &OrderBookResponse{}
	return rsp, t.client.Call("stock", "OrderBook", request, rsp)
}

// Get the last price for a given stock ticker
func (t *StockService) Price(request *PriceRequest) (*PriceResponse, error) {
	rsp := &PriceResponse{}
	return rsp, t.client.Call("stock", "Price", request, rsp)
}

// Get the last quote for the stock
func (t *StockService) Quote(request *QuoteRequest) (*QuoteResponse, error) {
	rsp := &QuoteResponse{}
	return rsp, t.client.Call("stock", "Quote", request, rsp)
}

type HistoryRequest struct {
	// date to retrieve as YYYY-MM-DD
	Date string `json:"date"`
	// the stock symbol e.g AAPL
	Stock string `json:"stock"`
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
	// the stock symbol
	Symbol string `json:"symbol"`
	// the volume
	Volume int32 `json:"volume"`
}

type Order struct {
	// the asking price
	AskPrice float64 `json:"askPrice"`
	// the ask size
	AskSize int32 `json:"askSize"`
	// the bidding price
	BidPrice float64 `json:"bidPrice"`
	// the bid size
	BidSize int32 `json:"bidSize"`
	// the UTC timestamp of the quote
	Timestamp string `json:"timestamp"`
}

type OrderBookRequest struct {
	// the date in format YYYY-MM-dd
	Date string `json:"date"`
	// optional RFC3339Nano end time e.g 2006-01-02T15:04:05.999999999Z07:00
	End string `json:"end"`
	// limit number of prices
	Limit int32 `json:"limit"`
	// optional RFC3339Nano start time e.g 2006-01-02T15:04:05.999999999Z07:00
	Start string `json:"start"`
	// stock to retrieve e.g AAPL
	Stock string `json:"stock"`
}

type OrderBookResponse struct {
	// date of the request
	Date string `json:"date"`
	// list of orders
	Orders []Order `json:"orders"`
	// the stock symbol
	Symbol string `json:"symbol"`
}

type PriceRequest struct {
	// stock symbol e.g AAPL
	Symbol string `json:"symbol"`
}

type PriceResponse struct {
	// the last price
	Price float64 `json:"price"`
	// the stock symbol e.g AAPL
	Symbol string `json:"symbol"`
}

type QuoteRequest struct {
	// the stock symbol e.g AAPL
	Symbol string `json:"symbol"`
}

type QuoteResponse struct {
	// the asking price
	AskPrice float64 `json:"askPrice"`
	// the ask size
	AskSize int32 `json:"askSize"`
	// the bidding price
	BidPrice float64 `json:"bidPrice"`
	// the bid size
	BidSize int32 `json:"bidSize"`
	// the stock symbol
	Symbol string `json:"symbol"`
	// the UTC timestamp of the quote
	Timestamp string `json:"timestamp"`
}

package nft

import (
	"go-micro.dev/v4/api/client"
)

type Nft interface {
	Assets(*AssetsRequest) (*AssetsResponse, error)
	Collections(*CollectionsRequest) (*CollectionsResponse, error)
	Create(*CreateRequest) (*CreateResponse, error)
}

func NewNftService(token string) *NftService {
	return &NftService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type NftService struct {
	client *client.Client
}

// Return a list of assets
func (t *NftService) Assets(request *AssetsRequest) (*AssetsResponse, error) {

	rsp := &AssetsResponse{}
	return rsp, t.client.Call("nft", "Assets", request, rsp)

}

// Get a list of collections
func (t *NftService) Collections(request *CollectionsRequest) (*CollectionsResponse, error) {

	rsp := &CollectionsResponse{}
	return rsp, t.client.Call("nft", "Collections", request, rsp)

}

// Create your own NFT (coming soon)
func (t *NftService) Create(request *CreateRequest) (*CreateResponse, error) {

	rsp := &CreateResponse{}
	return rsp, t.client.Call("nft", "Create", request, rsp)

}

type Asset struct {
	// associated collection
	Collection *Collection `json:"collection"`
	// asset contract
	Contract *Contract `json:"contract"`
	// Creator of the NFT
	Creator *User `json:"creator"`
	// related description
	Description string `json:"description"`
	// id of the asset
	Id int32 `json:"id"`
	// the image url
	ImageUrl string `json:"image_url"`
	// last time sold
	LastSale *Sale `json:"last_sale"`
	// listing date
	ListingDate string `json:"listing_date"`
	// name of the asset
	Name string `json:"name"`
	// Owner of the NFT
	Owner *User `json:"owner"`
	// the permalink
	Permalink string `json:"permalink"`
	// is it a presale
	Presale bool `json:"presale"`
	// number of sales
	Sales int32 `json:"sales"`
	// the token id
	TokenId string `json:"token_id"`
}

type AssetsRequest struct {
	// limit to members of a collection by slug name (case sensitive)
	Collection string `json:"collection"`
	// limit returned assets
	Limit int32 `json:"limit"`
	// offset for pagination
	Offset int32 `json:"offset"`
	// order "asc" or "desc"
	Order string `json:"order"`
	// order by "sale_date", "sale_count", "sale_price", "total_price"
	OrderBy string `json:"order_by"`
}

type AssetsResponse struct {
	// list of assets
	Assets []Asset `json:"assets"`
}

type Collection struct {
	CreatedAt     string `json:"created_at"`
	Description   string `json:"description"`
	ImageUrl      string `json:"image_url"`
	Name          string `json:"name"`
	PayoutAddress string `json:"payout_address"`
	Slug          string `json:"slug"`
}

type CollectionsRequest struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type CollectionsResponse struct {
	Collections []Collection `json:"collections"`
}

type Contract struct {
	// ethereum address
	Address string `json:"address"`
	// timestamp of creation
	CreatedAt string `json:"created_at"`
	// description of contract
	Description string `json:"description"`
	// name of contract
	Name string `json:"name"`
	// owner id
	Owner int32 `json:"owner"`
	// payout address
	PayoutAddress string `json:"payout_address"`
	// aka "ERC1155"
	Schema string `json:"schema"`
	// seller fees
	SellerFees string `json:"seller_fees"`
	// related symbol
	Symbol string `json:"symbol"`
	// type of contract e.g "semi-fungible"
	Type string `json:"type"`
}

type CreateRequest struct {
	// data if not image
	Data string `json:"data"`
	// description
	Description string `json:"description"`
	// image data
	Image string `json:"image"`
	// name of the NFT
	Name string `json:"name"`
}

type CreateResponse struct {
	Asset *Asset `json:"asset"`
}

type Sale struct {
	AssetDecimals  int32        `json:"asset_decimals"`
	AssetTokenId   string       `json:"asset_token_id"`
	CreatedAt      string       `json:"created_at"`
	EventTimestamp string       `json:"event_timestamp"`
	EventType      string       `json:"event_type"`
	PaymentToken   *Token       `json:"payment_token"`
	Quantity       string       `json:"quantity"`
	TotalPrice     string       `json:"total_price"`
	Transaction    *Transaction `json:"transaction"`
}

type Token struct {
	Address  string `json:"address"`
	Decimals int32  `json:"decimals"`
	EthPrice string `json:"eth_price"`
	Id       int32  `json:"id"`
	ImageUrl string `json:"image_url"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	UsdPrice string `json:"usd_price"`
}

type Transaction struct {
	BlockHash        string `json:"block_hash"`
	BlockNumber      string `json:"block_number"`
	FromAccount      *User  `json:"from_account"`
	Id               int32  `json:"id"`
	Timestamp        string `json:"timestamp"`
	ToAccount        *User  `json:"to_account"`
	TransactionHash  string `json:"transaction_hash"`
	TransactionIndex string `json:"transaction_index"`
}

type User struct {
	Address    string `json:"address"`
	ProfileUrl string `json:"profile_url"`
	Username   string `json:"username"`
}

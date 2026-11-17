package nano

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
)

// Network and Scheme are the identifiers this package offers in a 402.
const (
	Network = "nano:mainnet"
	Scheme  = "exact"
	Asset   = "XNO"
)

var orderIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// ErrInvalidOrderID is returned for an order id that is not 32 lowercase hex
// characters (128 random bits).
var ErrInvalidOrderID = errors.New("nano: invalid order id")

// NewOrderID returns a random 128-bit order id as lowercase hex. The order id
// is the payer's claim secret: it is never published on the ledger, only its
// hash (inside the representative commitment) is.
func NewOrderID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// ValidOrderID reports whether id has the shape NewOrderID produces.
func ValidOrderID(id string) bool { return orderIDPattern.MatchString(id) }

// Commitment binds one order to one destination and one price:
// SHA-256("xno-binding/v1/rep|" + orderID + "|" + payTo + "|" + priceRaw).
func Commitment(orderID, payTo, priceRaw string) [32]byte {
	return sha256.Sum256([]byte("xno-binding/v1/rep|" + orderID + "|" + payTo + "|" + priceRaw))
}

// CommitmentAccount is the Commitment encoded as a nano_ account. The payer
// sets it as the representative of the send block, so the signed block
// itself names the order it pays for.
func CommitmentAccount(orderID, payTo, priceRaw string) string {
	c := Commitment(orderID, payTo, priceRaw)
	acct, _ := EncodeAccount(c[:]) // 32 bytes, cannot fail
	return acct
}

// Offer is the nano:mainnet entry a 402 challenge carries.
type Offer struct {
	Scheme         string `json:"scheme"`
	Network        string `json:"network"`
	Asset          string `json:"asset"`
	PayTo          string `json:"payTo"`
	Amount         string `json:"amount"`
	OrderID        string `json:"orderId"`
	Representative string `json:"representative"`
}

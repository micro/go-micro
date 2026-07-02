package x402

import "strings"

// assetInfo describes a known stablecoin: its contract address and EIP-712
// domain (name, version) used to build a transfer-authorization signature.
type assetInfo struct {
	Address string
	Name    string
	Version string
}

// knownAssets maps a CAIP-2 network to its default stablecoin (USDC). Used to
// fill in the asset and its EIP-712 domain when the operator does not specify
// them, so a client can sign without hand-configured token metadata.
var knownAssets = map[string]assetInfo{
	"eip155:8453":  {"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", "USD Coin", "2"}, // Base mainnet
	"eip155:84532": {"0x036CbD53842c5426634e7929541eC2318f3dCF7e", "USDC", "2"},     // Base Sepolia
}

// NormalizeNetwork maps common short chain names to their CAIP-2 identifiers,
// which hosted facilitators (Coinbase CDP) use, and passes anything else
// through unchanged. Empty defaults to Base mainnet.
func NormalizeNetwork(n string) string {
	switch strings.ToLower(strings.TrimSpace(n)) {
	case "", "base", "eip155:8453":
		return "eip155:8453"
	case "base-sepolia", "eip155:84532":
		return "eip155:84532"
	default:
		return n
	}
}

func defaultAsset(network string) (assetInfo, bool) {
	a, ok := knownAssets[network]
	return a, ok
}

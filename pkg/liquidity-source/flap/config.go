package flap

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

// Config is populated per-chain from Kyber's external config. APIKey should only ever be injected via
// server-side config (never committed) since it grants access to the board API.
type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainID"`

	// PortalAddress is the flap.sh bonding-curve Portal proxy used for buy/sell and on-chain state
	// reads (getTokenV5, previewBuy, previewSell, ...).
	PortalAddress string `json:"portalAddress"`

	// APIBaseURL is the per-chain board API host, e.g. https://bnb.taxed.fun for BNB chain.
	APIBaseURL string `json:"apiBaseURL"`
	// APIKey is sent as the `trust-wallet-by-pass` header. Server-side only.
	APIKey string `json:"apiKey"`
}

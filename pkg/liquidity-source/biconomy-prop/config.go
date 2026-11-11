package biconomyprop

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Venue is the PropAMMVenue, 0x0000009E5F98cDB4dd1E1d2E15B220217c88B3b8 on Base.
	Venue string `json:"venue"`
	// HTTP is the Biconomy board feed (GET /venue/boards, x-api-key). The tracker reads it first
	// and falls back to on-chain reads when it errors or is stale. Unset: on-chain only.
	HTTP *HTTPConfig `json:"http,omitempty"`
	// MaxAgeMs rejects a feed response older than this (builtAtMs). Default 3000: measured REST
	// age p90 1.8s, max 2.2s.
	MaxAgeMs int64 `json:"maxAgeMs,omitempty"`
}

type HTTPConfig struct {
	BaseURL string            `json:"base_url"`
	Headers map[string]string `json:"headers"`
}

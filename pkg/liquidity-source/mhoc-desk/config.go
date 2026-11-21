package mhocdesk

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

// Config is the per-dex-id configuration. There is no factory or registry contract: each pool is one PropAMM
// contract deployed for one BASE/QUOTE pair, so the pool list is the configured address list.
type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`

	// Pools are PropAMM contract addresses, one per BASE/QUOTE pair. The list updater reads BASE(), QUOTE(),
	// PRICE_SCALE(), K_SCALE() and the tokens' decimals() from each one.
	Pools []string `json:"pools"`

	// Registry is the maker's PropAMMRegistry (optional). When set, the list updater also emits every pool that
	// registry.pools() returns, so a new pair is picked up without a configuration change.
	Registry string `json:"registry"`
}

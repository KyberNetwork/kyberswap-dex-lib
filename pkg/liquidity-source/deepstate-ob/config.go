package deepstateob

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config lists the fixed DeepstateV1 markets for this chain. Pools are
// permissionless on-chain but there is no factory/creation event to scan, so
// this integration tracks a static, config-seeded set of pairs instead of
// discovering them.
type Config struct {
	DexID   string              `json:"dexID"`
	ChainID valueobject.ChainID `json:"chainId"`

	// Router is the singleton DeepstateV1 matching-engine address. Every
	// pool in Pairs is served by this one contract.
	Router string `json:"router"`

	// Lens is the optional deployed book-reader periphery contract used to
	// batch the radix-tree BFS walk into one call per side. Empty until the
	// lens contract is deployed; the tracker falls back to a naive
	// per-node tree() BFS walk (see pool_tracker.go) when unset.
	Lens string `json:"lens"`

	// Pairs is the fixed set of token pairs to track, e.g. NVDA/USDG. Order
	// within each pair does not matter -- the tracker sorts by address to
	// match DeepstateV1's own token0/token1 convention.
	Pairs []PairConfig `json:"pairs"`
}

type PairConfig struct {
	TokenA string `json:"tokenA"`
	TokenB string `json:"tokenB"`
}

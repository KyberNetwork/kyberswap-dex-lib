package everlongclamm

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Config describes one Everlong CLAMM deployment (one CL pool whose sole LP is the
// Everlong ALM). One config entry per chain.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainID"`
	// PoolManager is the Infinity CLPoolManager singleton holding the pool.
	PoolManager string `json:"poolManager"`
	// ALM is the ClammALM proxy: the pool's sole LP, exposing the rung ladder
	// (getRungs) and the directional output haircut (poolFee).
	ALM string `json:"alm"`
	// Router is the ClammSwapRouter — the approval and call target for swaps.
	Router string `json:"router"`
	// PoolID is the bytes32 Infinity pool id = keccak256(abi.encode(PoolKey)).
	PoolID string `json:"poolId"`
	// Hook is PoolKey.hooks (the ClammHook enforcing exact-input + output haircut).
	Hook string `json:"hook"`
	// Fee is PoolKey.fee (uint24; the hook overrides the in-pool LP fee to 0 regardless).
	Fee uint32 `json:"fee"`
	// Parameters is PoolKey.parameters (bytes32 hex; packs hook permissions and, in
	// bits [16,40), the tick spacing).
	Parameters string `json:"parameters"`
	// Currency0/Currency1 are PoolKey.currency0/currency1 (sorted token addresses).
	Currency0 string `json:"currency0"`
	Currency1 string `json:"currency1"`
}

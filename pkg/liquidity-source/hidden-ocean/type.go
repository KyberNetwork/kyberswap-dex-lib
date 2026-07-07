package hiddenocean

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// Extra contains mutable pool state, JSON-marshaled into entity.Pool.Extra.
type Extra struct {
	SqrtPriceX96 *uint256.Int `json:"sqrtPriceX96"`
	Liquidity    *uint256.Int `json:"liquidity"`
	Fee          uint32       `json:"fee"`
	SqrtPaX96    *uint256.Int `json:"sqrtPaX96"`
	SqrtPbX96    *uint256.Int `json:"sqrtPbX96"`
}

// Metadata tracks incremental pool discovery progress via offset-based pagination.
type Metadata struct {
	Offset int `json:"offset"`
}

// RegistryPoolInfo mirrors the PoolInfo struct returned by HiddenOceanRegistry.getPool().
type RegistryPoolInfo struct {
	Pool   common.Address
	Token0 common.Address
	Token1 common.Address
}

// Slot0 mirrors the return values of pool.slot0().
type Slot0 struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Unlocked     bool
}

// RangeInfo mirrors the return values of pool.getRange().
type RangeInfo struct {
	SqrtPaX96 *big.Int
	SqrtPbX96 *big.Int
}

// SwapInfo is passed to the router for execution.
type SwapInfo struct{}

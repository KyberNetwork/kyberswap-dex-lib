package hiddenocean

import (
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

// SwapInfo is passed to the router for execution.
type SwapInfo struct{}

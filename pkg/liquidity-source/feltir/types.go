package feltir

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Extra struct {
	Samples [2][][2]*big.Int `json:"samples"` // [tokenInIndex][]{amountIn, amountOut}
}

type StaticExtra struct {
	FeltirAddress string `json:"feltirAddress"`
}

type PoolMetaInfo struct {
	BlockNumber   uint64 `json:"blockNumber"`
	FeltirAddress string `json:"feltirAddress"`
}

type PoolRPC struct {
	PoolId *big.Int       `json:"poolId"`
	Token0 common.Address `json:"token0"`
	Token1 common.Address `json:"token1"`
}

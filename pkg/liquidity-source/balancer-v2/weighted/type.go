package weighted

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Extra struct {
	SwapFeePercentage *big.Int   `json:"swapFeePercentage"`
	ScalingFactors    []*big.Int `json:"scalingFactors"`
	Paused            bool       `json:"paused"`
}

type StaticExtra struct {
	PoolID          string `json:"poolId"`
	PoolType        string `json:"poolType"`
	PoolTypeVersion int    `json:"poolTypeVersion"`
}

type PoolTokens struct {
	Tokens          []common.Address
	Balances        []*big.Int
	LastChangeBlock *big.Int
}

type Metadata struct {
	LastCreateTime *big.Int `json:"lastCreateTime"`
}

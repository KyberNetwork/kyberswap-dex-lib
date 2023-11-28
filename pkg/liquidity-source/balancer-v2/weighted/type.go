package weighted

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Extra struct {
	SwapFeePercentage *big.Int `json:"swapFeePercentage"`
	Paused            bool     `json:"paused"`
}

type StaticExtra struct {
	PoolID          string     `json:"poolId"`
	PoolType        string     `json:"poolType"`
	PoolTypeVersion int        `json:"poolTypeVersion"`
	ScalingFactors  []*big.Int `json:"scalingFactors"`
}

type PoolTokens struct {
	Tokens          []common.Address
	Balances        []*big.Int
	LastChangeBlock *big.Int
}

type PausedState struct {
	Paused              bool
	PauseWindowEndTime  *big.Int
	BufferPeriodEndTime *big.Int
}

type rpcRes struct {
	PoolTokens        PoolTokens
	SwapFeePercentage *big.Int
	PausedState       PausedState
	BlockNumber       uint64
}

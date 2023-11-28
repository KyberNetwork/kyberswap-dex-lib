package stable

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Gas struct {
	Swap int64
}

type Extra struct {
	Amp               *big.Int `json:"amp"`
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

type AmplificationParameter struct {
	Value      *big.Int
	IsUpdating bool
	Precision  *big.Int
}

type PoolMetaInfo struct {
	T string `json:"t"`
	V int    `json:"v"`
}

type rpcRes struct {
	Amp               *big.Int
	PoolTokens        PoolTokens
	SwapFeePercentage *big.Int
	PausedState       PausedState
	BlockNumber       uint64
}

package weighted

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Extra struct {
	SwapFeePercentage *uint256.Int `json:"swapFeePercentage"`
	Paused            bool         `json:"paused"`
}

type StaticExtra struct {
	PoolID            string         `json:"poolId"`
	PoolType          string         `json:"poolType"`
	PoolTypeVer       int            `json:"poolTypeVer"`
	ScalingFactors    []*uint256.Int `json:"scalingFactors"`
	NormalizedWeights []*uint256.Int `json:"normalizedWeights"`
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

type PoolMetaInfo struct {
	T string `json:"t"`
	V int    `json:"v"`
}

type rpcRes struct {
	PoolTokens        PoolTokens
	SwapFeePercentage *uint256.Int
	PausedState       PausedState
	BlockNumber       uint64
}

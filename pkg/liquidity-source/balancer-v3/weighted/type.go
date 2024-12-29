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
	PoolID         string         `json:"poolId"`
	PoolType       string         `json:"poolType"`
	PoolVersion    int            `json:"poolVersion"`
	ScalingFactors []*uint256.Int `json:"scalingFactors"`
	Vault          string         `json:"vault"`
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
	Vault         string `json:"vault"`
	PoolID        string `json:"poolId"`
	TokenOutIndex int    `json:"tokenOutIndex"`
	BlockNumber   uint64 `json:"blockNumber"`
}

type rpcRes struct {
	PoolTokens        PoolTokens
	SwapFeePercentage *uint256.Int
	PausedState       PausedState
	BlockNumber       uint64
}

package gyro2clp

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Gas struct {
	Swap int64
}

type Extra struct {
	SwapFeePercentage *uint256.Int `json:"swapFeePercentage"`
	Paused            bool         `json:"paused"`
}

type StaticExtra struct {
	PoolID         string         `json:"poolId"`
	PoolType       string         `json:"poolType"`
	PoolTypeVer    int            `json:"poolTypeVersion"`
	ScalingFactors []*uint256.Int `json:"scalingFactors"`
	SqrtParameters []*uint256.Int `json:"sqrtParameters"`
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
	Vault           string `json:"vault"`
	PoolID          string `json:"poolId"`
	TokenOutIndex   int    `json:"tokenOutIndex"`
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress"`
}

type rpcRes struct {
	PoolTokens        PoolTokens
	SwapFeePercentage *big.Int
	PausedState       PausedState
	BlockNumber       uint64
}

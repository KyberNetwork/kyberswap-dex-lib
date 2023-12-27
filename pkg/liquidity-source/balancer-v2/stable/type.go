package stable

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Gas struct {
	Swap int64
}

type Extra struct {
	Amp               *uint256.Int   `json:"amp"`
	SwapFeePercentage *uint256.Int   `json:"swapFeePercentage"`
	ScalingFactors    []*uint256.Int `json:"scalingFactors"`
	Paused            bool           `json:"paused"`
}

type StaticExtra struct {
	PoolID             string `json:"poolId"`
	PoolType           string `json:"poolType"`
	PoolTypeVer        int    `json:"poolTypeVersion"`
	PoolSpecialization uint8  `json:"poolSpecialization"`
	Vault              string `json:"vault"`
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
	Vault         string `json:"vault"`
	PoolID        string `json:"poolId"`
	TokenOutIndex int    `json:"tokenOutIndex"`
	BlockNumber   uint64 `json:"blockNumber"`
}

type rpcRes struct {
	Amp               *big.Int
	PoolTokens        PoolTokens
	SwapFeePercentage *big.Int
	ScalingFactors    []*big.Int
	PausedState       PausedState
	BlockNumber       uint64
}

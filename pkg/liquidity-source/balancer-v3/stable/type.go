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
	AmplificationParameter     *uint256.Int   `json:"amplificationParameter"`
	SwapFeePercentage          *uint256.Int   `json:"swapFeePercentage"`
	AggregateSwapFeePercentage *uint256.Int   `json:"aggregateSwapFeePercentage"`
	BalancesLiveScaled18       []*uint256.Int `json:"balancesLiveScaled18"`
	DecimalScalingFactors      []*uint256.Int `json:"decimalScalingFactors"`
	TokenRates                 []*uint256.Int `json:"tokenRates"`
	IsPaused                   bool           `json:"isPaused"`
	IsVaultLocked              bool           `json:"isVaultLocked"`
}

type StaticExtra struct {
	PoolType    string `json:"poolType"`
	PoolVersion int    `json:"poolVersion"`
	Vault       string `json:"vault"`
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
	PoolType      string `json:"poolType"`
	PoolVersion   int    `json:"poolVersion"`
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

type SwapInfo struct {
	AggregateFee *big.Int `json:"aggregateFee"`
}

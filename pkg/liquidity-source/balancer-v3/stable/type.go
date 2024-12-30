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
	StaticSwapFeePercentage    *uint256.Int   `json:"staticSwapFeePercentage"`
	AggregateSwapFeePercentage *uint256.Int   `json:"aggregateSwapFeePercentage"`
	BalancesLiveScaled18       []*uint256.Int `json:"balancesLiveScaled18"`
	DecimalScalingFactors      []*uint256.Int `json:"decimalScalingFactors"`
	TokenRates                 []*uint256.Int `json:"tokenRates"`
	IsPaused                   bool           `json:"isPaused"`
	IsInRecoveryMode           bool           `json:"isInRecoveryMode"`
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

type AggregateFeePercentage struct {
	AggregateSwapFeePercentage  *big.Int
	AggregateYieldFeePercentage *big.Int
}

type StablePoolDynamicData struct {
	Data struct {
		BalancesLiveScaled18    []*big.Int
		TokenRates              []*big.Int
		StaticSwapFeePercentage *big.Int
		TotalSupply             *big.Int
		BptRate                 *big.Int
		AmplificationParameter  *big.Int
		StartValue              *big.Int
		EndValue                *big.Int
		StartTime               uint32
		EndTime                 uint32
		IsAmpUpdating           bool
		IsPoolInitialized       bool
		IsPoolPaused            bool
		IsPoolInRecoveryMode    bool
	}
}

type PoolTokenInfo struct {
	Tokens                   []common.Address
	TokenInfo                []TokenInfo
	BalancesRaw              []*big.Int
	LastBalancesLiveScaled18 []*big.Int
}

type TokenInfo struct {
	TokenType     uint8
	IRateProvider common.Address
	PaysYieldFees bool
}

type PoolMetaInfo struct {
	Vault         string `json:"vault"`
	PoolType      string `json:"poolType"`
	PoolVersion   int    `json:"poolVersion"`
	TokenOutIndex int    `json:"tokenOutIndex"`
	BlockNumber   uint64 `json:"blockNumber"`
}

type RpcResult struct {
	BalancesRaw                []*big.Int
	BalancesLiveScaled18       []*big.Int
	TokenRates                 []*big.Int
	StaticSwapFeePercentage    *big.Int
	AggregateSwapFeePercentage *big.Int
	AmplificationParameter     *big.Int
	IsPoolPaused               bool
	IsPoolInRecoveryMode       bool
	BlockNumber                uint64
}

type SwapInfo struct {
	AggregateFee *big.Int `json:"aggregateFee"`
}

package shared

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type StaticExtra struct {
	Hook         string   `json:"hook,omitempty"`
	HookType     HookType `json:"hookT,omitempty"`
	BufferTokens []string `json:"buffs,omitempty"`
}

type Extra struct {
	HooksConfig                `json:"hook"`
	StaticSwapFeePercentage    *uint256.Int   `json:"fee,omitempty"`
	AggregateSwapFeePercentage *uint256.Int   `json:"aggrFee,omitempty"`
	BalancesLiveScaled18       []*uint256.Int `json:"balsE18,omitempty"`
	DecimalScalingFactors      []*uint256.Int `json:"decs,omitempty"`
	TokenRates                 []*uint256.Int `json:"rates,omitempty"`
	Buffers                    []*ExtraBuffer `json:"buffs,omitempty"`
}

type RpcResult struct {
	HooksConfigRPC
	StaticSwapFeePercentage *big.Int
	AggregateFeePercentageRPC
	PoolDataRPC
	Buffers        []*ExtraBufferRPC
	IsPoolDisabled bool
	BlockNumber    uint64
}

type ExtraBuffer struct {
	TotalAssets *uint256.Int `json:"tA,omitempty"`
	TotalSupply *uint256.Int `json:"tS,omitempty"`
}

type PoolDataRPC struct {
	PoolData struct {
		PoolConfigBits        [32]byte
		Tokens                []common.Address
		TokenInfo             []TokenInfo
		BalancesRaw           []*big.Int
		BalancesLiveScaled18  []*big.Int
		TokenRates            []*big.Int
		DecimalScalingFactors []*big.Int
	}
}

type ExtraBufferRPC struct {
	TotalAssets *big.Int
	TotalSupply *big.Int
}

type HooksConfig struct {
	EnableHookAdjustedAmounts       bool `json:"adjAmts,omitempty"`
	ShouldCallComputeDynamicSwapFee bool `json:"dynFee,omitempty"`
	ShouldCallBeforeSwap            bool `json:"befSwap,omitempty"`
	ShouldCallAfterSwap             bool `json:"aftSwap,omitempty"`
}

type HooksConfigRPC struct {
	HooksConfigData struct {
		EnableHookAdjustedAmounts       bool
		ShouldCallBeforeInitialize      bool
		ShouldCallAfterInitialize       bool
		ShouldCallComputeDynamicSwapFee bool
		ShouldCallBeforeSwap            bool
		ShouldCallAfterSwap             bool
		ShouldCallBeforeAddLiquidity    bool
		ShouldCallAfterAddLiquidity     bool
		ShouldCallBeforeRemoveLiquidity bool
		ShouldCallAfterRemoveLiquidity  bool
		HooksContract                   common.Address
	}
}

type SwapInfo struct {
	Buffers      []*ExtraBuffer
	AggregateFee *big.Int
}

type PoolMetaInfo struct {
	BufferTokenIn  string `json:"buffIn"`
	BufferTokenOut string `json:"buffOut"`
}

type AggregateFeePercentageRPC struct {
	AggregateSwapFeePercentage  *big.Int
	AggregateYieldFeePercentage *big.Int
}

type VaultSwapParams struct {
	Kind           SwapKind
	IndexIn        int
	IndexOut       int
	AmountGivenRaw *uint256.Int
}

type PoolSwapParams struct {
	Kind                    SwapKind
	OnSwap                  OnSwapFn
	StaticSwapFeePercentage *uint256.Int
	AmountGivenScaled18     *uint256.Int
	BalancesScaled18        []*uint256.Int
	IndexIn                 int
	IndexOut                int
}

type OnSwapFn func(param PoolSwapParams) (*uint256.Int, error)

type AfterSwapParams struct {
	Kind                     SwapKind
	IndexIn                  int
	IndexOut                 int
	AmountInScaled18         *uint256.Int
	AmountOutScaled18        *uint256.Int
	TokenInBalanceScaled18   *uint256.Int
	TokenOutBalanceScaled18  *uint256.Int
	AmountCalculatedScaled18 *uint256.Int
	AmountCalculatedRaw      *uint256.Int
}

type TokenInfo struct {
	TokenType     uint8
	RateProvider  common.Address
	PaysYieldFees bool
}

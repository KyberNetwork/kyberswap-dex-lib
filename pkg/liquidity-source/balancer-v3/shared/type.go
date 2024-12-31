package shared

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type AggregateFeePercentage struct {
	AggregateSwapFeePercentage  *big.Int
	AggregateYieldFeePercentage *big.Int
}

type VaultSwapParams struct {
	Kind           SwapKind
	IndexIn        int
	IndexOut       int
	AmountGivenRaw *uint256.Int
	LimitRaw       *uint256.Int
	// HooksConfig                HooksConfig
	// DecimalScalingFactor       *uint256.Int
	// TokenRate                  *uint256.Int
	// AmplificationParameter     *uint256.Int
	// SwapFeePercentage          *uint256.Int
	// AggregateSwapFeePercentage *uint256.Int
	// BalancesLiveScaled18       []*uint256.Int
}

type PoolSwapParams struct {
	Kind                 SwapKind
	SwapFeePercentage    *uint256.Int
	AmountGivenScaled18  *uint256.Int
	BalancesLiveScaled18 []*uint256.Int
	IndexIn              int
	IndexOut             int
}

type TokenInfo struct {
	TokenType     uint8
	RateProvider  common.Address
	PaysYieldFees bool
}

type PoolDataRPC struct {
	Data struct {
		PoolConfigBits        [32]byte
		Tokens                []common.Address
		TokenInfo             []TokenInfo
		BalancesRaw           []*big.Int
		BalancesLiveScaled18  []*big.Int
		TokenRates            []*big.Int
		DecimalScalingFactors []*big.Int
	}
}

type HooksConfig struct {
	ShouldCallComputeDynamicSwapFee bool `json:"shouldCallComputeDynamicSwapFee"`
	ShouldCallBeforeSwap            bool `json:"shouldCallBeforeSwap"`
	ShouldCallAfterSwap             bool `json:"shouldCallAfterSwap"`
}

type HooksConfigRPC struct {
	Data struct {
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

package types

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	mapset "github.com/deckarep/golang-set/v2"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type AggregateParams struct {
	TokenIn  entity.SimplifiedToken
	TokenOut entity.SimplifiedToken
	GasToken entity.SimplifiedToken

	TokenInPriceUSD  float64
	TokenOutPriceUSD float64
	GasTokenPriceUSD float64

	// AmountIn amount of tokenIn
	AmountIn *big.Int

	AmountInUsd float64

	// Sources list of liquidity sources to be finding route on
	Sources []string

	// OnlySinglePath
	//	- if true: finds single path route only
	//	- if false: finds single path route and multi path route then return the better one
	OnlySinglePath bool

	// GasInclude
	// 	- if true: better route has more (amountOutUSD - gasUSD)
	//  - if false: better route return more amount of tokenOut
	GasInclude bool

	// GasPrice price of gas
	GasPrice *big.Float

	// L1FeeOverhead
	L1FeeOverhead *big.Int

	// L1FeePerPool
	L1FeePerPool *big.Int

	// ExtraFee fee charged by client
	ExtraFee valueobject.ExtraFee

	// IsHillClimbEnabled use hill climb finder to adjust split amountIn to get better amountOut
	IsHillClimbEnabled bool

	// By default, we will use nativeTvlIndex
	// If feature flag IsLiquidityScoreIndexEnabled enable, combined tvl + liquidity score will be used, otherwise we will use liquidity score index
	Index valueobject.IndexType

	// ExcludedPools name of pool addresses are excluded when finding route, separated by comma
	ExcludedPools mapset.Set[string]

	ClientId string

	// KyberLimitOrderAllowedSenders is a comma-separated list of addresses used to filter
	// Kyber private limit orders.
	KyberLimitOrderAllowedSenders string

	// Flag to enable alpha fee reduction
	EnableAlphaFee bool

	// Flag to enable hill climbing for amm best route
	EnableHillClaimForAlphaFee bool

	IsScaleHelperClient bool
}

type AggregateBundledParamsPair struct {
	TokenIn  string
	TokenOut string

	// AmountIn amount of tokenIn
	AmountIn    *big.Int
	AmountInUsd float64
}

type AggregateBundledParams struct {
	GasToken string

	// Sources list of liquidity sources to be finding route on
	Sources []string

	// GasInclude
	// 	- if true: better route has more (amountOutUSD - gasUSD)
	//  - if false: better route return more amount of tokenOut
	GasInclude bool

	// GasPrice price of gas
	GasPrice *big.Float

	// L1FeeOverhead
	L1FeeOverhead *big.Int

	// L1FeePerPool
	L1FeePerPool *big.Int

	// IsHillClimbEnabled use hill climb finder to adjust split amountIn to get better amountOut
	IsHillClimbEnabled bool

	// By default, we will use nativeTvlIndex
	// If feature flag IsLiquidityScoreIndexEnabled enable, combined tvl + liquidity score will be used, otherwise we will use liquidity score index
	Index valueobject.IndexType

	// ExcludedPools name of pool addresses are excluded when finding route, separated by comma
	ExcludedPools mapset.Set[string]

	ClientId string

	Pairs []AggregateBundledParamsPair

	OverridePools []*entity.Pool

	// ExtraWhitelistedTokens list of token addresses are included in whitelisted when finding route
	ExtraWhitelistedTokens []string

	KyberLimitOrderAllowedSenders string

	IsScaleHelperClient bool
}

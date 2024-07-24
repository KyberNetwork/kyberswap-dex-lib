package types

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	mapset "github.com/deckarep/golang-set/v2"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type AggregateParams struct {
	TokenIn  entity.Token
	TokenOut entity.Token
	GasToken entity.Token

	TokenInPriceUSD  float64
	TokenOutPriceUSD float64
	GasTokenPriceUSD float64

	// AmountIn amount of tokenIn
	AmountIn *big.Int

	// Sources list of liquidity sources to be finding route on
	Sources []string

	// SaveGas
	//	- if true: finds single path route only
	//	- if false: finds single path route and multi path route then return the better one
	SaveGas bool

	// GasInclude
	// 	- if true: better route has more (amountOutUSD - gasUSD)
	//  - if false: better route return more amount of tokenOut
	GasInclude bool

	// GasPrice price of gas
	GasPrice *big.Float

	// ExtraFee fee charged by client
	ExtraFee valueobject.ExtraFee

	// IsHillClimbEnabled use hill climb finder to adjust split amountIn to get better amountOut
	IsHillClimbEnabled bool

	// ExcludedPools name of pool addresses are excluded when finding route, separated by comma
	ExcludedPools mapset.Set[string]

	ClientId string
}

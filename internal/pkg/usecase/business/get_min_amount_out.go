package business

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// GetMinAmountOutFunc returns minimum possible amountOut
type GetMinAmountOutFunc func(outputAmount *big.Int, slippageTolerance *big.Int) *big.Int

// GetMinAmountOutExactInput returns minimum possible amountOut in exact input trade type
// outputAmount - outputAmount * slippageTolerance / basicPoint
// https://team-kyber.slack.com/archives/C03M949J2JX/p1666714423459349
func GetMinAmountOutExactInput(outputAmount *big.Int, slippageTolerance *big.Int) *big.Int {
	numerator := new(big.Int).Mul(outputAmount, slippageTolerance)
	res := new(big.Int).Sub(outputAmount, new(big.Int).Div(numerator, valueobject.BasisPoint))
	if new(big.Int).Mod(numerator, valueobject.BasisPoint).Cmp(big.NewInt(0)) == 0 {
		return res
	}
	return new(big.Int).Sub(res, big.NewInt(1))
}

// GetMinAmountOutExactOutput returns minimum possible amountOut in exact output trade type
// outputAmount
func GetMinAmountOutExactOutput(outputAmount *big.Int, _ *big.Int) *big.Int {
	return outputAmount
}

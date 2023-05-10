package business

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// GetMinAmountOutFunc returns minimum possible amountOut
type GetMinAmountOutFunc func(outputAmount *big.Int, slippageTolerance *big.Int) *big.Int

// GetMinAmountOutExactInput returns minimum possible amountOut in exact input trade type
// (outputAmount * basisPoint) / (slippageTolerance + basisPoint)
func GetMinAmountOutExactInput(outputAmount *big.Int, slippageTolerance *big.Int) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Mul(
			outputAmount,
			valueobject.BasisPoint,
		),
		new(big.Int).Add(
			slippageTolerance,
			valueobject.BasisPoint,
		),
	)
}

// GetMinAmountOutExactOutput returns minimum possible amountOut in exact output trade type
// outputAmount
func GetMinAmountOutExactOutput(outputAmount *big.Int, _ *big.Int) *big.Int {
	return outputAmount
}

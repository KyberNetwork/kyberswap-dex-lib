package core

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// GetMaxAmountInFunc returns maximum possible amountIn
type GetMaxAmountInFunc func(inputAmount *big.Int, slippageTolerance *big.Int) *big.Int

// GetMaxAmountInExactInput returns maximum possible amountIn in exact input trade type
// inputAmount
func GetMaxAmountInExactInput(inputAmount *big.Int, _ *big.Int) *big.Int {
	return inputAmount
}

// GetMaxAmountInExactOutput returns maximum possible amountIn in exact output trade type
// ((basisPoint + slippageTolerance) * inputAmount) / basisPoint
func GetMaxAmountInExactOutput(inputAmount *big.Int, slippageTolerance *big.Int) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Mul(
			new(big.Int).Add(valueobject.BasisPoint, slippageTolerance),
			inputAmount,
		),
		valueobject.BasisPoint,
	)
}

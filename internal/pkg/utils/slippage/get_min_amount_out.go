package slippage

import (
	"math/big"

	"github.com/shopspring/decimal"
)

const (
	BasisPoint = 10000
)

// GetMinAmountOutFunc returns minimum possible amountOut
type GetMinAmountOutFunc func(outputAmount *big.Int, slippageTolerance float64) *big.Int

// GetMinAmountOutExactInput returns minimum possible amountOut in exact input trade type
// outputAmount - outputAmount * slippageTolerance / basicPoint
// https://team-kyber.slack.com/archives/C03M949J2JX/p1666714423459349
func GetMinAmountOutExactInput(outputAmount *big.Int, slippageTolerance float64) *big.Int {
	res := decimal.NewFromBigInt(outputAmount, 0).
		Mul(decimal.NewFromFloat(1 - slippageTolerance/BasisPoint)).
		BigInt()
	if res.Sign() <= 0 {
		res.SetInt64(1)
	}
	return res
}

// GetMinAmountOutExactOutput returns minimum possible amountOut in exact output trade type
// outputAmount
func GetMinAmountOutExactOutput(outputAmount *big.Int, _ float64) *big.Int {
	return outputAmount
}

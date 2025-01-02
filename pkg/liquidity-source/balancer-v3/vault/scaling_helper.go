package vault

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/math"
	"github.com/holiman/uint256"
)

func toScaled18ApplyRateRoundUp(amount, scalingFactor, tokenRate *uint256.Int) (*uint256.Int, error) {
	scaledAmount, err := math.FixPoint.Mul(amount, scalingFactor)
	if err != nil {
		return nil, err
	}
	return math.FixPoint.MulUp(scaledAmount, tokenRate)
}

func toScaled18ApplyRateRoundDown(amount, scalingFactor, tokenRate *uint256.Int) (*uint256.Int, error) {
	scaledAmount, err := math.FixPoint.Mul(amount, scalingFactor)
	if err != nil {
		return nil, err
	}
	return math.FixPoint.MulDown(scaledAmount, tokenRate)
}

func computeRateRoundUp(rate *uint256.Int) *uint256.Int {
	divisor := new(uint256.Int).Div(rate, math.ONE_E18)
	divisor.Mul(divisor, math.ONE_E18)

	if divisor.Eq(rate) {
		return divisor.Set(rate)
	}

	return divisor.Add(rate, math.ONE)
}

func toRawUndoRateRoundDown(amount, scalingFactor, tokenRate *uint256.Int) (*uint256.Int, error) {
	divisor, err := math.FixPoint.Mul(scalingFactor, tokenRate)
	if err != nil {
		return nil, err
	}

	return math.FixPoint.DivDown(amount, divisor)
}

// func toRawUndoRateRoundUp(amount, scalingFactor, tokenRate *uint256.Int) (*uint256.Int, error) {
// 	divisor, err := math.FixPoint.Mul(scalingFactor, tokenRate)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return math.FixPoint.DivUp(amount, divisor)
// }

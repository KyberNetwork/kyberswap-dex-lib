package math

import (
	"math"
	"math/big"
)

var (
	MaxSqrtRatio = IntFromString("6276949602062853172742588666638147158083741740262337144812")
	MinSqrtRatio = IntFromString("18447191164202170526")
)

func ToSqrtRatio(tick int32) *big.Int {
	ratio := new(big.Int).Set(oneX128)

	var tickAbs int32
	if tick < 0 {
		tickAbs = -tick
	} else {
		tickAbs = tick
	}

	for i, mask := range tickMasks {
		if tickAbs&(1<<i) != 0 {
			ratio.Rsh(
				ratio.Mul(ratio, mask),
				128,
			)
		}
	}

	if tick > 0 {
		ratio.Div(U256Max, ratio)
	}

	return ratio
}

func u256ToFloatBaseX128(x128 *big.Int) float64 {
	bf := new(big.Float).SetInt(x128)
	scale := new(big.Float).SetInt(TwoPow128)

	bf.Quo(bf, scale)

	result, _ := bf.Float64()
	return result
}

func ApproximateNumberOfTickSpacingsCrossed(startingSqrtRatio, endingSqrtRatio *big.Int, tickSpacing uint32) uint32 {
	if tickSpacing == 0 {
		return 0
	}

	start, end := u256ToFloatBaseX128(startingSqrtRatio), u256ToFloatBaseX128(endingSqrtRatio)
	ticksCrossed := uint32(math.Abs(math.Log(start)-math.Log(end)) / logBaseSqrtTickSize)

	return ticksCrossed / tickSpacing
}

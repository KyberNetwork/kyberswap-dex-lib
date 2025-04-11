package math

import (
	"math"
	"math/big"
)

const (
	MinTick int32 = -88722835
	MaxTick int32 = 88722835
)

var (
	MaxSqrtRatio = IntFromString("6276949602062853172742588666607187473671941430179807625216")
	MinSqrtRatio = IntFromString("18447191164202170524")
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

	if ratio.Cmp(TwoPow160) != -1 {
		ratio.Rsh(ratio, 98).Lsh(ratio, 98)
	} else if ratio.Cmp(TwoPow128) != -1 {
		ratio.Rsh(ratio, 66).Lsh(ratio, 66)
	} else if ratio.Cmp(TwoPow96) != -1 {
		ratio.Rsh(ratio, 34).Lsh(ratio, 34)
	} else {
		ratio.Rsh(ratio, 2).Lsh(ratio, 2)
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

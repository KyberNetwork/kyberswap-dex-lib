package math

import (
	"math"

	"github.com/KyberNetwork/kutils"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	MinTick int32 = -88722835
	MaxTick int32 = 88722835
)

var (
	MaxSqrtRatio = big256.New("6276949602062853172742588666607187473671941430179807625216")
	MinSqrtRatio = big256.New("18447191164202170524")
)

func ToSqrtRatio(tick int32) *uint256.Int {
	ratio := big256.U2Pow128.Clone()

	tickAbs := kutils.Abs(tick)
	for i, mask := range tickMasks {
		if tickAbs&(1<<i) != 0 {
			ratio.Rsh(
				ratio.Mul(ratio, mask),
				128,
			)
		}
	}

	if tick > 0 {
		ratio.Div(big256.UMax, ratio)
	}

	if bitLen := ratio.BitLen(); bitLen > 160 {
		ratio.Rsh(ratio, 98).Lsh(ratio, 98)
	} else if bitLen > 128 {
		ratio.Rsh(ratio, 66).Lsh(ratio, 66)
	} else if bitLen > 96 {
		ratio.Rsh(ratio, 34).Lsh(ratio, 34)
	} else {
		ratio.Rsh(ratio, 2).Lsh(ratio, 2)
	}

	return ratio
}

func ApproximateNumberOfTickSpacingsCrossed(startingSqrtRatio, endingSqrtRatio *uint256.Int, tickSpacing uint32) uint32 {
	if tickSpacing == 0 {
		return 0
	}

	start, end := U256ToFloatBaseX128(startingSqrtRatio), U256ToFloatBaseX128(endingSqrtRatio)
	ticksCrossed := uint32(math.Abs(math.Log(start/end)) / logBaseSqrtTickSize)

	return ticksCrossed / tickSpacing
}

func ApproximateSqrtRatioToTick(sqrtRatio *uint256.Int) int32 {
	return int32(math.Round(math.Log(U256ToFloatBaseX128(sqrtRatio)) / logBaseSqrtTickSize))
}

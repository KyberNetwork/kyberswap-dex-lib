package math

import (
	"math"
	"math/big"

	"github.com/KyberNetwork/kutils"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	MinTick int32 = -88722835
	MaxTick int32 = 88722835
)

var (
	MaxSqrtRatio = bignum.NewBig("6276949602062853172742588666607187473671941430179807625216")
	MinSqrtRatio = bignum.NewBig("18447191164202170524")
)

func ToSqrtRatio(tick int32) *big.Int {
	ratio := new(big.Int).Set(TwoPow128)

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
		ratio.Div(U256Max, ratio)
	}

	bitLen := ratio.BitLen()
	if bitLen > 160 {
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

func ApproximateNumberOfTickSpacingsCrossed(startingSqrtRatio, endingSqrtRatio *big.Int, tickSpacing uint32) uint32 {
	if tickSpacing == 0 {
		return 0
	}

	start, _ := startingSqrtRatio.Float64()
	end, _ := endingSqrtRatio.Float64()
	ticksCrossed := uint32(math.Abs(math.Log(start/end)) / logBaseSqrtTickSize)

	return ticksCrossed / tickSpacing
}

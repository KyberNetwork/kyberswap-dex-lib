package math

import (
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	bitMask    = uint256.MustFromHex("0xc00000000000000000000000")
	notBitMask = uint256.MustFromHex("0x3fffffffffffffffffffffff")
)

// FloatSqrtRatio
// A dynamic fixed point number (a la floating point) that stores a shifting 94 bit view of the underlying fixed point
// value, based on the most significant bits (mantissa)
// If the most significant 2 bits are 11, it represents a 64.30
// If the most significant 2 bits are 10, it represents a 32.62 number
// If the most significant 2 bits are 01, it represents a 0.94 number
// If the most significant 2 bits are 00, it represents a 0.126 number that is always less than 2**-32

func FloatSqrtRatioToFixed(sqrtRatioFloat *uint256.Int) *uint256.Int {
	var tmp uint256.Int
	op2 := tmp.Rsh(tmp.And(sqrtRatioFloat, bitMask), 89).Uint64() + 2
	op1 := tmp.And(sqrtRatioFloat, notBitMask)
	return op1.Lsh(op1, uint(op2))
}

func FixedSqrtRatioToFloat(sqrtRatioFixed *uint256.Int, roundUp bool) *uint256.Int {
	var tmp uint256.Int
	if added := lo.IfF(roundUp, func() *uint256.Int { return tmp.AddUint64(sqrtRatioFixed, 3) }).
		Else(tmp.Set(sqrtRatioFixed)); added.Lt(big256.U2Pow96) {
		return added.Rsh(added, 2)
	}

	if added := lo.IfF(roundUp, func() *uint256.Int { return tmp.Add(sqrtRatioFixed, big256.UMaxU34) }).
		Else(tmp.Set(sqrtRatioFixed)); added.Lt(big256.U2Pow128) {
		return added.Rsh(added, 34).Or(added, big256.U2Pow94)
	}

	if added := lo.IfF(roundUp, func() *uint256.Int { return tmp.Add(sqrtRatioFixed, big256.UMaxU66) }).
		Else(tmp.Set(sqrtRatioFixed)); added.Lt(big256.U2Pow160) {
		return added.Rsh(added, 66).Or(added, big256.U2Pow95)
	}

	added := lo.IfF(roundUp, func() *uint256.Int { return tmp.Add(sqrtRatioFixed, big256.UMaxU98) }).
		Else(tmp.Set(sqrtRatioFixed))
	return added.Rsh(added, 98).Or(added, bitMask)
}

func nextSqrtRatioFromAmount0(sqrtRatio, liquidity, amount0 *uint256.Int) (*uint256.Int, error) {
	if amount0.IsZero() {
		return sqrtRatio.Clone(), nil
	} else if liquidity.IsZero() {
		return nil, ErrNoLiquidity
	}

	var num, tmp uint256.Int
	num.Lsh(liquidity, 128)

	if amount0.Sign() < 0 {
		amount0Abs := tmp.Neg(amount0)
		product, overflow := amount0Abs.MulOverflow(amount0Abs, sqrtRatio)
		if overflow {
			return nil, ErrOverflow
		}

		denominator, underflow := product.SubOverflow(&num, product)
		if underflow {
			return nil, ErrUnderflow
		}

		return MulDivOverflow(&num, sqrtRatio, denominator, true)
	} else {
		denomP1 := tmp.Div(&num, sqrtRatio)

		denom, overflow := denomP1.AddOverflow(denomP1, amount0)
		if overflow {
			return nil, ErrOverflow
		}

		return div(&num, denom, true)
	}
}

func nextSqrtRatioFromAmount1(sqrtRatio, liquidity, amount1 *uint256.Int) (*uint256.Int, error) {
	if amount1.IsZero() {
		return sqrtRatio.Clone(), nil
	} else if liquidity.IsZero() {
		return nil, ErrNoLiquidity
	}

	var tmp uint256.Int
	amount1Abs := tmp.Abs(amount1)
	roundUp := amount1.Sign() < 0

	quotient, err := MulDivOverflow(amount1Abs, big256.U2Pow128, liquidity, roundUp)
	if err != nil {
		return nil, err
	}

	if roundUp {
		res, overflow := quotient.SubOverflow(sqrtRatio, quotient)
		if overflow {
			return nil, ErrUnderflow
		}
		return res, nil
	} else {
		res, overflow := quotient.AddOverflow(sqrtRatio, quotient)
		if overflow {
			return nil, ErrOverflow
		}
		return res, nil
	}
}

var sqrtRatioRegions = [...]struct {
	shift uint
	bound *uint256.Int
}{
	{2, big256.U2Pow96},
	{34, big256.U2Pow128},
	{66, big256.U2Pow160},
	{98, big256.U2Pow192},
}

// RoundSqrtRatio rounds a fixed point sqrt ratio to the precision Core stores it with, in the given direction, like
// `toSqrtRatio` in the contracts: the region is chosen after rounding, so rounding up can move a ratio into the next,
// coarser region. Ratios too large to be stored round to the maximum uint256.
func RoundSqrtRatio(sqrtRatio *uint256.Int, roundUp bool) *uint256.Int {
	var rounded uint256.Int
	for _, region := range sqrtRatioRegions {
		rounded.Set(sqrtRatio)
		if roundUp {
			var mask uint256.Int
			mask.Lsh(big256.U1, region.shift).SubUint64(&mask, 1)
			if _, overflow := rounded.AddOverflow(&rounded, &mask); overflow {
				rounded.SetAllOne()
			}
		}
		if rounded.Lt(region.bound) {
			return rounded.Lsh(rounded.Rsh(&rounded, region.shift), region.shift)
		}
	}
	return rounded.SetAllOne()
}

// IsValidSqrtRatio reports whether a fixed point sqrt ratio is a sqrt ratio Core can store and swap to.
func IsValidSqrtRatio(sqrtRatio *uint256.Int) bool {
	return !sqrtRatio.Lt(MinSqrtRatio) && !sqrtRatio.Gt(MaxSqrtRatio) && RoundSqrtRatio(sqrtRatio, false).Eq(sqrtRatio)
}

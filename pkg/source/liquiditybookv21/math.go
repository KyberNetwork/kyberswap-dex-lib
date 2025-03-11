package liquiditybookv21

import (
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func getPriceFromID(id uint32, binStep uint16) (*uint256.Int, error) {
	var base, exponent uint256.Int
	getBase(binStep, &base)
	getExponent(id, &exponent)
	return pow(&base, &exponent)
}

func getBase(binStep uint16, base *uint256.Int) *uint256.Int {
	u := uint256.NewInt(uint64(binStep))
	u = u.Lsh(u, scaleOffset)
	return base.Add(scale, u.Div(u, uBasisPointMax))
}

func getExponent(id uint32, exponent *uint256.Int) *uint256.Int {
	return exponent.SetUint64(uint64(id)).SubUint64(exponent, realIDShift)
}

// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/math/Uint128x128Math.sol#L95
func pow(x, y *uint256.Int) (*uint256.Int, error) {
	if y.IsZero() {
		return scale, nil
	}

	var absY, result, squared, tmp uint256.Int
	absY.Abs(y)
	invert := y.Sign() < 0

	if absY.Cmp(powU) < 0 {
		result.Set(scale)
		squared.Set(x)

		if x.Cmp(big256.TwoPow128) >= 0 {
			squared.Div(big256.Max, &squared)
			invert = !invert
		}

		for i := 0x1; i <= 0x80000; i <<= 1 {
			if !tmp.And(&absY, tmp.SetUint64(uint64(i))).IsZero() {
				result.Rsh(tmp.Mul(&result, &squared), 128)
			}
			if i < 0x80000 {
				squared.Rsh(tmp.Mul(&squared, &squared), 128)
			}
		}
	}

	if result.IsZero() {
		return nil, ErrPowUnderflow
	}

	if invert {
		result.Div(big256.Max, &result)
	}

	return &result, nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L95
func shiftDivRoundUp(x *uint256.Int, offset uint8, denominator *uint256.Int) (*uint256.Int, error) {
	result, err := shiftDivRoundDown(x, offset, denominator)
	if err != nil {
		return nil, err
	}

	// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L97
	// mulmod(x, y, m): (x * y) % m with arbitrary precision arithmetic, 0 if m == 0

	if denominator.IsZero() {
		return new(uint256.Int), nil
	}
	var v uint256.Int
	if !v.MulMod(
		x, v.Lsh(big256.One, uint(offset)),
		denominator,
	).IsZero() {
		result.AddUint64(result, 1)
	}

	return result, nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L114
func shiftDivRoundDown(x *uint256.Int, offset uint8, denominator *uint256.Int) (*uint256.Int, error) {
	var prod0, prod1, y uint256.Int
	prod0.Lsh(x, uint(offset))
	prod1.Rsh(x, uint(256-int(offset)))
	y.Lsh(big256.One, uint(offset))
	return getEndOfDivRoundDown(x, &y, denominator, &prod0, &prod1)
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L172
func getEndOfDivRoundDown(
	x *uint256.Int,
	y *uint256.Int,
	denominator *uint256.Int,
	prod0 *uint256.Int,
	prod1 *uint256.Int,
) (*uint256.Int, error) {
	if prod1.IsZero() {
		return new(uint256.Int).Div(prod0, denominator), nil
	}

	if prod1.Cmp(denominator) >= 0 {
		return nil, ErrMulDivOverflow
	}

	var remainder uint256.Int
	if remainder.MulMod(x, y, denominator).Cmp(prod0) > 0 {
		prod1 = prod1.SubUint64(prod1, 1)
	}
	prod0 = prod0.Sub(prod0, &remainder)

	// bitwiseNotDenominator = ~denominator, denominator has type uint256
	var lpotdod uint256.Int
	lpotdod.And(denominator, lpotdod.Neg(denominator))

	denominator = denominator.Div(denominator, &lpotdod)

	prod0 = prod0.Div(prod0, &lpotdod)

	// // Flip lpotdod such that it is 2^256 / lpotdod. If lpotdod is zero, then it becomes one
	// lpotdod := add(div(sub(0, lpotdod), lpotdod), 1)
	lpotdod.AddUint64(lpotdod.Div(lpotdod.Neg(&lpotdod), &lpotdod), 1)

	tmp := remainder
	prod0 = prod0.Or(prod0, tmp.Mul(prod1, &lpotdod))

	inverse := prod1.Mul(prod1.Mul(denominator, denominator), big256.U9)

	for range 6 {
		inverse.Mul(
			inverse,
			tmp.Sub(big256.Two, tmp.Mul(denominator, inverse)),
		)
	}

	result := prod0.Mul(prod0, inverse)
	return result, nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L95
func mulShiftRoundUp(x, y *uint256.Int, offset uint8) (*uint256.Int, error) {
	result, err := mulShiftRoundDown(x, y, offset)
	if err != nil {
		return nil, err
	}
	var v uint256.Int
	if !v.MulMod(
		x, y,
		v.Lsh(big256.One, uint(offset)),
	).IsZero() {
		result.AddUint64(result, 1)
	}
	return result, nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L67
func mulShiftRoundDown(x, y *uint256.Int, offset uint8) (*uint256.Int, error) {
	var prod0, prod1 uint256.Int
	getMulProds(x, y, &prod0, &prod1)
	result := new(uint256.Int)
	if !prod0.IsZero() {
		result.Rsh(&prod0, uint(offset))
	}
	if !prod1.IsZero() {
		var tmp uint256.Int
		if prod1.Cmp(tmp.Lsh(big256.One, uint(offset))) >= 0 {
			return nil, ErrMulShiftOverflow
		}
		result.Add(
			result,
			tmp.Lsh(&prod1, 256-uint(offset)),
		)
	}

	return result, nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L152
func getMulProds(x, y *uint256.Int, prod0, prod1 *uint256.Int) (*uint256.Int, *uint256.Int) {
	var mm uint256.Int
	mm.MulMod(x, y, big256.Max)
	prod0.Mul(x, y)
	prod1.Sub(&mm, prod0)
	if mm.Cmp(prod0) < 0 {
		prod1.SubUint64(prod1, 1)
	}
	return prod0, prod1
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/FeeHelper.sol#L58
func getFeeAmount(amount, totalFee *uint256.Int) (*uint256.Int, error) {
	if err := verifyFee(totalFee); err != nil {
		return nil, err
	}

	var result, denominator uint256.Int
	return result.Div(
		result.SubUint64(
			result.Add(
				result.Mul(amount, totalFee),
				denominator.Sub(precision, totalFee),
			),
			1,
		),
		&denominator,
	), nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/FeeHelper.sol#L40
func getFeeAmountFrom(amountWithFees, totalFee *uint256.Int) (*uint256.Int, error) {
	if err := verifyFee(totalFee); err != nil {
		return nil, err
	}

	var result uint256.Int
	return result.Div(
		result.SubUint64(
			result.Add(
				result.Mul(amountWithFees, totalFee),
				precision,
			),
			1,
		),
		precision,
	), nil
}

func verifyFee(fee *uint256.Int) error {
	if fee.Cmp(maxFee) > 0 {
		return ErrFeeTooLarge
	}
	return nil
}

func scalarMulDivBasisPointRoundDown(totalFee, multiplier *uint256.Int) (*uint256.Int, error) {
	if multiplier.IsZero() {
		return new(uint256.Int), nil
	}

	if multiplier.Cmp(uBasisPointMax) > 0 {
		return nil, ErrMultiplierTooLarge
	}

	result := new(uint256.Int)
	return result.Div(
		result.Mul(totalFee, multiplier),
		uBasisPointMax,
	), nil
}

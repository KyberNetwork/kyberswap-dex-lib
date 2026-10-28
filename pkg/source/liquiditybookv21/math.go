package liquiditybookv21

import (
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// The helpers below write their result into z, which must not be one of their inputs, and leave the
// inputs as they were.

func getPriceFromID(z *uint256.Int, id uint32, binStep uint16) error {
	var base, exponent uint256.Int
	getBase(binStep, &base)
	getExponent(id, &exponent)
	return pow(z, &base, &exponent)
}

func getBase(binStep uint16, base *uint256.Int) *uint256.Int {
	var u uint256.Int
	u.SetUint64(uint64(binStep))
	u.Lsh(&u, scaleOffset)
	return base.Add(scale, u.Div(&u, uBasisPointMax))
}

func getExponent(id uint32, exponent *uint256.Int) *uint256.Int {
	return exponent.SetUint64(uint64(id)).SubUint64(exponent, realIDShift)
}

// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/math/Uint128x128Math.sol#L95
func pow(z, x, y *uint256.Int) error {
	if y.IsZero() {
		z.Set(scale)
		return nil
	}

	var absY, squared, tmp uint256.Int
	absY.Abs(y)
	invert := y.Sign() < 0

	z.Clear()
	if absY.Cmp(powU) < 0 {
		z.Set(scale)
		squared.Set(x)

		if x.Cmp(big256.U2Pow128) >= 0 {
			squared.Div(big256.UMax, &squared)
			invert = !invert
		}

		for i := 0x1; i <= 0x80000; i <<= 1 {
			if absY[0]&uint64(i) != 0 {
				z.Rsh(tmp.Mul(z, &squared), 128)
			}
			if i < 0x80000 {
				squared.Rsh(tmp.Mul(&squared, &squared), 128)
			}
		}
	}

	if z.IsZero() {
		return ErrPowUnderflow
	}

	if invert {
		z.Div(big256.UMax, tmp.Set(z))
	}

	return nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L95
func shiftDivRoundUp(z, x *uint256.Int, offset uint8, denominator *uint256.Int) error {
	if err := shiftDivRoundDown(z, x, offset, denominator); err != nil {
		return err
	}

	// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L97
	// mulmod(x, y, m): (x * y) % m with arbitrary precision arithmetic, 0 if m == 0

	if denominator.IsZero() {
		z.Clear()
		return nil
	}
	var v uint256.Int
	if !v.MulMod(x, v.Lsh(big256.U1, uint(offset)), denominator).IsZero() {
		z.AddUint64(z, 1)
	}

	return nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L114
func shiftDivRoundDown(z, x *uint256.Int, offset uint8, denominator *uint256.Int) error {
	var prod0, prod1, y uint256.Int
	prod0.Lsh(x, uint(offset))
	prod1.Rsh(x, uint(256-int(offset)))
	y.Lsh(big256.U1, uint(offset))
	return getEndOfDivRoundDown(z, x, &y, denominator, &prod0, &prod1)
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L172
// It works on prod0 and prod1, which are the caller's scratch.
func getEndOfDivRoundDown(z, x, y, denominator, prod0, prod1 *uint256.Int) error {
	if prod1.IsZero() {
		z.Div(prod0, denominator)
		return nil
	}

	if prod1.Cmp(denominator) >= 0 {
		return ErrMulDivOverflow
	}

	var remainder uint256.Int
	if remainder.MulMod(x, y, denominator).Cmp(prod0) > 0 {
		prod1.SubUint64(prod1, 1)
	}
	prod0.Sub(prod0, &remainder)

	// bitwiseNotDenominator = ~denominator, denominator has type uint256
	var lpotdod, d uint256.Int
	lpotdod.And(denominator, lpotdod.Neg(denominator))

	d.Div(denominator, &lpotdod)

	prod0.Div(prod0, &lpotdod)

	// Flip lpotdod such that it is 2^256 / lpotdod. If lpotdod is zero, then it becomes one
	// lpotdod := add(div(sub(0, lpotdod), lpotdod), 1)
	var negLpotdod uint256.Int
	lpotdod.AddUint64(lpotdod.Div(negLpotdod.Neg(&lpotdod), &lpotdod), 1)

	tmp := remainder
	prod0.Or(prod0, tmp.Mul(prod1, &lpotdod))

	// inverse := (3 * denominator) ^ 2, where ^ is xor: the inverse modulo 2^4 to start Newton from
	var inverse uint256.Int
	inverse.Xor(inverse.Mul(&d, big256.U3), big256.U2)

	for range 6 {
		inverse.Mul(&inverse, tmp.Sub(big256.U2, tmp.Mul(&d, &inverse)))
	}

	z.Mul(prod0, &inverse)
	return nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L95
func mulShiftRoundUp(z, x, y *uint256.Int, offset uint8) error {
	if err := mulShiftRoundDown(z, x, y, offset); err != nil {
		return err
	}
	var v uint256.Int
	if !v.MulMod(x, y, v.Lsh(big256.U1, uint(offset))).IsZero() {
		z.AddUint64(z, 1)
	}
	return nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L67
func mulShiftRoundDown(z, x, y *uint256.Int, offset uint8) error {
	var prod0, prod1 uint256.Int
	getMulProds(x, y, &prod0, &prod1)
	z.Clear()
	if !prod0.IsZero() {
		z.Rsh(&prod0, uint(offset))
	}
	if !prod1.IsZero() {
		var tmp uint256.Int
		if prod1.Cmp(tmp.Lsh(big256.U1, uint(offset))) >= 0 {
			return ErrMulShiftOverflow
		}
		z.Add(z, tmp.Lsh(&prod1, 256-uint(offset)))
	}

	return nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L152
func getMulProds(x, y *uint256.Int, prod0, prod1 *uint256.Int) (*uint256.Int, *uint256.Int) {
	var mm uint256.Int
	mm.MulMod(x, y, big256.UMax)
	prod0.Mul(x, y)
	prod1.Sub(&mm, prod0)
	if mm.Cmp(prod0) < 0 {
		prod1.SubUint64(prod1, 1)
	}
	return prod0, prod1
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/FeeHelper.sol#L58
func getFeeAmount(z, amount, totalFee *uint256.Int) error {
	if err := verifyFee(totalFee); err != nil {
		return err
	}

	var denominator uint256.Int
	denominator.Sub(precision, totalFee)
	z.Div(z.SubUint64(z.Add(z.Mul(amount, totalFee), &denominator), 1), &denominator)
	return nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/FeeHelper.sol#L40
func getFeeAmountFrom(z, amountWithFees, totalFee *uint256.Int) error {
	if err := verifyFee(totalFee); err != nil {
		return err
	}

	z.Div(z.SubUint64(z.Add(z.Mul(amountWithFees, totalFee), precision), 1), precision)
	return nil
}

func verifyFee(fee *uint256.Int) error {
	if fee.Cmp(maxFee) > 0 {
		return ErrFeeTooLarge
	}
	return nil
}

func scalarMulDivBasisPointRoundDown(z, totalFee, multiplier *uint256.Int) error {
	if multiplier.IsZero() {
		z.Clear()
		return nil
	}

	if multiplier.Cmp(uBasisPointMax) > 0 {
		return ErrMultiplierTooLarge
	}

	z.Div(z.Mul(totalFee, multiplier), uBasisPointMax)
	return nil
}

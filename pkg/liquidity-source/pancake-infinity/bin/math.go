package bin

import (
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Functions reused from the logic of liquidity-book-v21

func getPriceFromID(id uint32, binStep uint16) (*uint256.Int, error) {
	var base, exponent uint256.Int
	getBase(binStep, &base)
	getExponent(id, &exponent)
	return pow(&base, &exponent)
}

func getBase(binStep uint16, base *uint256.Int) *uint256.Int {
	u := uint256.NewInt(uint64(binStep))
	u = u.Lsh(u, _SCALE_OFFSET)
	return base.Add(_SCALE, u.Div(u, _BASIS_POINT_MAX))
}

func getExponent(id uint32, exponent *uint256.Int) *uint256.Int {
	return exponent.SetUint64(uint64(id)).SubUint64(exponent, _REAL_ID_SHIFT)
}

func pow(x, y *uint256.Int) (*uint256.Int, error) {
	if y.IsZero() {
		return _SCALE, nil
	}

	var absY, result, squared, tmp uint256.Int
	absY.Abs(y)
	invert := y.Sign() < 0

	if absY.Cmp(_POW_U) < 0 {
		result.Set(_SCALE)
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

func shiftDivRoundUp(x *uint256.Int, offset uint8, denominator *uint256.Int) (*uint256.Int, error) {
	result, err := shiftDivRoundDown(x, offset, denominator)
	if err != nil {
		return nil, err
	}

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

func shiftDivRoundDown(x *uint256.Int, offset uint8, denominator *uint256.Int) (*uint256.Int, error) {
	var prod0, prod1, y uint256.Int
	prod0.Lsh(x, uint(offset))
	prod1.Rsh(x, uint(256-int(offset)))
	y.Lsh(big256.One, uint(offset))
	return getEndOfDivRoundDown(x, &y, denominator, &prod0, &prod1)
}

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

// Functions with custom logic from pancake-infinity-bin

func getFeeAmount(amount, feeBips *uint256.Int) *uint256.Int {
	totalFee := new(uint256.Int).Mul(feeBips, _ONE_E12)
	denominator := new(uint256.Int).Sub(_PRECISION, totalFee)

	res := totalFee.Mul(amount, totalFee)
	res.Add(res, denominator)
	res.SubUint64(res, 1)

	return res.Div(res, denominator)
}

func getFeeAmountFrom(amountWithFees, feeBips *uint256.Int) *uint256.Int {
	totalFee := new(uint256.Int).Mul(feeBips, _ONE_E12)
	totalFee.Mul(amountWithFees, totalFee)
	totalFee.Add(totalFee, _PRECISION)
	totalFee.SubUint64(totalFee, 1)

	return totalFee.Div(totalFee, _PRECISION)
}

func calculateSwapFee(protocolFee, lpFee *uint256.Int) *uint256.Int {
	fee1 := new(uint256.Int).And(protocolFee, _MASK12)
	fee2 := new(uint256.Int).And(lpFee, _MASK24)

	numerator := new(uint256.Int).Mul(fee1, fee2)
	quotient := numerator.Div(numerator, _PIPS_DENOMINATOR)
	sum := new(uint256.Int).Add(fee1, fee2)

	return sum.Sub(sum, quotient)
}

func getProtocolFeeAmt(amount, protocolFee, swapFee *uint256.Int) *uint256.Int {
	if protocolFee.IsZero() || swapFee.IsZero() {
		return amount.SetUint64(0)
	}

	if protocolFee.Eq(swapFee) {
		return amount
	}

	return new(uint256.Int).Div(amount.Mul(amount, protocolFee), swapFee)
}

func getZeroForOneFee(protocolFee *uint256.Int) *uint256.Int {
	return new(uint256.Int).And(protocolFee, _MASK12)
}

func getOneForZeroFee(protocolFee *uint256.Int) *uint256.Int {
	return new(uint256.Int).Rsh(protocolFee, 12)
}

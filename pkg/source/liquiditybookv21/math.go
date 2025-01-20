package liquiditybookv21

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func getPriceFromID(id uint32, binStep uint16) (*big.Int, error) {
	var base, exponent big.Int
	getBase(binStep, &base)
	getExponent(id, &exponent)
	return pow(&base, &exponent)
}

func getBase(binStep uint16, base *big.Int) *big.Int {
	u := new(big.Int).Lsh(big.NewInt(int64(binStep)), scaleOffset)
	return base.Add(scale, new(big.Int).Div(u, big.NewInt(basisPointMax)))
}

func getExponent(id uint32, exponent *big.Int) *big.Int {
	return exponent.Sub(big.NewInt(int64(id)), big.NewInt(realIDShift))
}

// https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/libraries/math/Uint128x128Math.sol#L95
func pow(x *big.Int, y *big.Int) (*big.Int, error) {
	var (
		invert  bool
		absY    big.Int
		result  big.Int
		squared big.Int
		tmp     big.Int
		and     big.Int
	)

	if y.Sign() == 0 {
		return scale, nil
	}

	absY.Abs(y)
	if y.Sign() < 0 {
		invert = !invert
	}

	var u, v big.Int
	u.SetString("100000", 16)
	v.SetString("ffffffffffffffffffffffffffffffff", 16)

	if absY.Cmp(&u) < 0 {
		result.Set(scale)
		squared.Set(x)

		if x.Cmp(&v) > 0 {
			squared.Div(bignumber.MAX_UINT_256, &squared)
			invert = !invert
		}

		for i := 0x1; i <= 0x80000; i <<= 1 {
			and.And(&absY, big.NewInt(int64(i)))
			if and.Sign() != 0 {
				result.Rsh(tmp.Mul(&result, &squared), 128)
			}
			if i < 0x80000 {
				squared.Rsh(tmp.Mul(&squared, &squared), 128)
			}
		}
	}

	if result.Sign() == 0 {
		return nil, ErrPowUnderflow
	}

	if invert {
		v.Sub(new(big.Int).Lsh(bignumber.One, 256), bignumber.One)
		result.Div(&v, &result)
	}

	return &result, nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L95
func shiftDivRoundUp(x *big.Int, offset uint8, denominator *big.Int) (*big.Int, error) {
	result, err := shiftDivRoundDown(x, offset, denominator)
	if err != nil {
		return nil, err
	}

	// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L97
	// mulmod(x, y, m): (x * y) % m with arbitrary precision arithmetic, 0 if m == 0

	if denominator.Sign() == 0 {
		return new(big.Int), nil
	}
	v := new(big.Int).Mod(
		new(big.Int).Mul(x, new(big.Int).Lsh(bignumber.One, uint(offset))),
		denominator,
	)
	if v.Sign() != 0 {
		result.Add(result, bignumber.One)
	}

	return result, nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L114
func shiftDivRoundDown(x *big.Int, offset uint8, denominator *big.Int) (*big.Int, error) {
	var (
		prod0, prod1 *big.Int
	)

	prod0 = new(big.Int).Lsh(x, uint(offset))
	prod1 = new(big.Int).Rsh(x, uint(256-int(offset)))

	y := new(big.Int).Lsh(bignumber.One, uint(offset))

	return getEndOfDivRoundDown(x, y, denominator, prod0, prod1)
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L172
func getEndOfDivRoundDown(
	x *big.Int,
	y *big.Int,
	denominator *big.Int,
	prod0 *big.Int,
	prod1 *big.Int,
) (*big.Int, error) {
	if prod1.Sign() == 0 {
		return new(big.Int).Div(prod0, denominator), nil
	}

	if prod1.Cmp(denominator) >= 0 {
		return nil, ErrMulDivOverflow
	}

	var remainder big.Int
	if denominator.Sign() != 0 {
		remainder.Mod(new(big.Int).Mul(x, y), denominator)
	}

	prod1 = new(big.Int).Sub(prod1, gt(&remainder, prod0))
	prod0 = new(big.Int).Sub(prod0, &remainder)

	// bitwiseNotDenominator = ~denominator, denominator has type uint256
	lpotdod := new(big.Int).And(denominator, new(big.Int).Add(bitwiseNotUint256(denominator), bignumber.One))

	denominator = new(big.Int).Div(denominator, lpotdod)

	prod0 = new(big.Int).Div(prod0, lpotdod)

	lpotdod = new(big.Int).Add(
		new(big.Int).Div(new(big.Int).Sub(bignumber.ZeroBI, lpotdod), lpotdod),
		bignumber.One,
	)

	prod0 = new(big.Int).Or(prod0, new(big.Int).Mul(prod1, lpotdod))

	inverse := new(big.Int).Mul(new(big.Int).Mul(denominator, denominator), big.NewInt(9))

	for i := 0; i < 6; i++ {
		inverse.Mul(
			inverse,
			new(big.Int).Sub(bignumber.Two, new(big.Int).Mul(denominator, inverse)),
		)
	}

	result := new(big.Int).Mul(prod0, inverse)
	return result, nil
}

func gt(x *big.Int, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return bignumber.One
	}
	return bignumber.ZeroBI
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L95
func mulShiftRoundUp(x *big.Int, y *big.Int, offset uint8) (*big.Int, error) {
	result, err := mulShiftRoundDown(x, y, offset)
	if err != nil {
		return nil, err
	}
	v := new(big.Int).Mod(
		new(big.Int).Mul(x, y),
		new(big.Int).Lsh(bignumber.One, uint(offset)),
	)
	if v.Sign() != 0 {
		result.Add(result, bignumber.One)
	}
	return result, nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L67
func mulShiftRoundDown(x *big.Int, y *big.Int, offset uint8) (*big.Int, error) {
	prod0, prod1 := new(big.Int), new(big.Int)
	getMulProds(x, y, prod0, prod1)
	result := new(big.Int)
	if prod0.Sign() != 0 {
		result.Rsh(prod0, uint(offset))
	}
	if prod1.Sign() != 0 {
		if prod1.Cmp(new(big.Int).Lsh(bignumber.One, uint(offset))) >= 0 {
			return nil, ErrMulShiftOverflow
		}
		result.Add(
			result,
			new(big.Int).Lsh(prod1, uint(256-int(offset))),
		)
	}

	return result, nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/math/Uint256x256Math.sol#L152
func getMulProds(x *big.Int, y *big.Int, prod0, prod1 *big.Int) (*big.Int, *big.Int) {
	mm := new(big.Int).Mod(new(big.Int).Mul(x, y), bignumber.MAX_UINT_256)
	prod0.Mul(x, y)
	prod1.Sub(
		new(big.Int).Sub(mm, prod0),
		lt(mm, prod0),
	)
	return prod0, prod1
}

func lt(x *big.Int, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return bignumber.One
	}
	return bignumber.ZeroBI
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/FeeHelper.sol#L58
func getFeeAmount(amount *big.Int, totalFee *big.Int) (*big.Int, error) {
	if err := verifyFee(totalFee); err != nil {
		return nil, err
	}

	denominator := new(big.Int).Sub(precison, totalFee)
	result := new(big.Int).Div(
		new(big.Int).Sub(
			new(big.Int).Add(
				new(big.Int).Mul(amount, totalFee),
				denominator,
			),
			bignumber.One,
		),
		denominator,
	)
	return result, nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/libraries/FeeHelper.sol#L40
func getFeeAmountFrom(amountWithFees *big.Int, totalFee *big.Int) (*big.Int, error) {
	if err := verifyFee(totalFee); err != nil {
		return nil, err
	}

	result := new(big.Int).Div(
		new(big.Int).Sub(
			new(big.Int).Add(
				new(big.Int).Mul(amountWithFees, totalFee),
				precison,
			),
			bignumber.One,
		),
		precison,
	)
	return result, nil
}

func bitwiseNotUint256(x *big.Int) *big.Int {
	return new(big.Int).Xor(x, bignumber.MAX_UINT_256)
}

func verifyFee(fee *big.Int) error {
	if fee.Cmp(maxFee) > 0 {
		return ErrFeeTooLarge
	}
	return nil
}

func scalarMulDivBasisPointRoundDown(totalFee *big.Int, multiplier *big.Int) (*big.Int, error) {
	if multiplier.Sign() == 0 {
		return new(big.Int), nil
	}

	if multiplier.Cmp(big.NewInt(basisPointMax)) > 0 {
		return nil, ErrMultiplierTooLarge
	}

	result := new(big.Int).Div(
		new(big.Int).Mul(totalFee, multiplier),
		big.NewInt(basisPointMax),
	)
	return result, nil
}

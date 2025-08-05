package math

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/int256"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

var (
	minX        = i256.MustFromDecimal("-42139678854452767551")
	maxX        = i256.MustFromDecimal("135305999368893231589")
	fiveToThe18 = i256.MustFromDecimal("3814697265625")
	ln2Scaled   = i256.MustFromDecimal("54916777467707473351141471128")
	twoTo95     = i256.MustFromDecimal("39614081257132168796771975168")

	p0 = i256.MustFromDecimal("1346386616545796478920950773328")
	p1 = i256.MustFromDecimal("57155421227552351082224309758442")
	p2 = i256.MustFromDecimal("94201549194550492254356042504812")
	p3 = i256.MustFromDecimal("28719021644029726153956944680412240")
	p4 = i256.MustFromDecimal("4385272521454847904659076985693276")

	q0 = i256.MustFromDecimal("2855989394907223263936484059900")
	q1 = i256.MustFromDecimal("50020603652535783019961831881945")
	q2 = i256.MustFromDecimal("533845033583426703283633433725380")
	q3 = i256.MustFromDecimal("3604857256930695427073651918091429")
	q4 = i256.MustFromDecimal("14423608567350463180887372962807573")
	q5 = i256.MustFromDecimal("26449188498355588339934803723976023")

	scaleFactor = uint256.MustFromDecimal("3822833074963236453042738258902158003155416615667")

	ErrOverflow = errors.New("overflow")
)

func MulDivUp(a, b, denominator *uint256.Int) *uint256.Int {
	res, _ := v3Utils.MulDivRoundingUp(a, b, denominator)
	return res
}

func DivUp(a, b *uint256.Int) *uint256.Int {
	return MulDivUp(a, u256.BONE, b)
}

func Rpow(x *uint256.Int, n int, base *uint256.Int) (*uint256.Int, error) {
	if x.IsZero() {
		if n == 0 {
			return base.Clone(), nil
		}

		return u256.U0.Clone(), nil
	}

	var (
		z       uint256.Int
		xx      uint256.Int
		xxRound uint256.Int
		zx      uint256.Int
		zxRound uint256.Int
		i       uint256.Int
		temp    uint256.Int
	)

	if n%2 == 0 {
		z.Set(base)
	}

	var half uint256.Int
	half.Div(base, u256.U2)
	for ; i.Sign() > 0; i.Div(&i, u256.U2) {
		xx.Mul(x, x)

		if !temp.Div(&xx, x).Eq(x) {
			return nil, ErrOverflow
		}

		xxRound.Add(&xx, &half)
		if xxRound.Lt(&xx) {
			return nil, ErrOverflow
		}

		x = new(uint256.Int).Div(&xxRound, base)
		if !temp.Mod(&i, u256.U2).IsZero() {
			zx.Mul(&z, x)

			if !x.IsZero() && !temp.Div(&zx, x).Eq(&z) {
				return nil, ErrOverflow
			}

			zxRound.Add(&zx, &half)
			if zxRound.Lt(&zx) {
				return nil, ErrOverflow
			}

			z.Div(&zxRound, base)
		}
	}

	return &z, nil
}

func Abs(x *int256.Int) *uint256.Int {
	var res int256.Int
	res.Set(x)

	if res.IsNegative() {
		res.Neg(&res)
	}

	return i256.SafeConvertToUInt256(&res)
}

func SubReLU(a, b *uint256.Int) *uint256.Int {
	if a.Lt(b) {
		return uint256.NewInt(0)
	}
	var result uint256.Int
	result.Sub(a, b)
	return &result
}

func Dist(x, y *uint256.Int) *uint256.Int {
	var z uint256.Int
	if x.Gt(y) {
		return z.Sub(x, y)
	}
	return z.Sub(y, x)
}

func MulWadUp(x, y *uint256.Int) (*uint256.Int, error) {
	if y.IsZero() {
		return uint256.NewInt(0), nil
	}

	var tmp uint256.Int
	tmp.SetAllOne()
	tmp.Div(&tmp, y)
	if x.Gt(&tmp) {
		return nil, errors.New("MulWadFailed")
	}

	var result uint256.Int
	result.Mul(x, y)

	tmp.Clear()
	result.DivMod(&result, WAD, &tmp)

	if !tmp.IsZero() {
		result.AddUint64(&result, 1)
	}

	return &result, nil
}

func ExpWad(x *int256.Int) (*uint256.Int, error) {
	/*
	   @dev Calculates the natural exponential function of a signed integer with
	        a precision of 1e18.
	   @notice Note that this function consumes about 810 gas units. The implementation
	           is inspired by Remco Bloemen's implementation under the MIT license here:
	           https://xn--2-umb.com/22/exp-ln.
	   @dev This implementation is derived from Snekmate, which is authored
	        by pcaversaccio (Snekmate), distributed under the AGPL-3.0 license.
	        https://github.com/pcaversaccio/snekmate
	   @param x The 32-byte variable.
	   @return int256 The 32-byte calculation result.
	*/

	// # If the result is `< 0.5`, we return zero. This happens when we have the following:
	// # "x <= floor(log(0.5e18) * 1e18) ~ -42e18".
	if x.Cmp(minX) <= 0 {
		return uint256.NewInt(0), nil
	}

	// # When the result is "> (2 ** 255 - 1) / 1e18" we cannot represent it as a signed integer.
	// # This happens when "x >= floor(log((2 ** 255 - 1) / 1e18) * 1e18) ~ 135".
	if x.Cmp(maxX) >= 0 {
		return nil, errors.New("WadExpOverflow")
	}

	// # `x` is now in the range "(-42, 136) * 1e18". Convert to "(-42, 136) * 2 ** 96" for higher
	// # intermediate precision and a binary base. This base conversion is a multiplication with
	// # "1e18 / 2 ** 96 = 5 ** 18 / 2 ** 78".
	value := i256.Div(i256.Lsh(x, 78), fiveToThe18)

	// # Reduce the range of `x` to "(-½ ln 2, ½ ln 2) * 2 ** 96" by factoring out powers of two
	// # so that "exp(x) = exp(x') * 2 ** k", where `k` is a signer integer. Solving this gives
	// # "k = round(x / log(2))" and "x' = x - k * log(2)". Thus, `k` is in the range "[-61, 195]".
	k := i256.Rsh(
		i256.Add(
			i256.Div(
				i256.Lsh(value, 96),
				ln2Scaled),
			twoTo95),
		96)
	value = i256.Sub(value, i256.Mul(k, ln2Scaled))

	// # Evaluate using a "(6, 7)"-term rational approximation. Since `p` is monic,
	// # we will multiply by a scaling factor later.
	y := i256.Add(
		i256.Rsh(
			i256.Mul(i256.Add(value, p0), value),
			96),
		p1)
	p := i256.Add(
		i256.Mul(
			i256.Add(
				i256.Rsh(
					i256.Mul(
						i256.Sub(i256.Add(y, value), p2),
						y),
					96),
				p3),
			value),
		i256.Lsh(p4, 96),
	)

	// # We leave `p` in the "2 ** 192" base so that we do not have to scale it up
	// # again for the division.
	q := i256.Add(
		i256.Rsh(
			i256.Mul(i256.Sub(value, q0), value),
			96),
		q1)
	q = i256.Sub(i256.Rsh(i256.Mul(q, value), 96), q2)
	q = i256.Add(i256.Rsh(i256.Mul(q, value), 96), q3)
	q = i256.Sub(i256.Rsh(i256.Mul(q, value), 96), q4)
	q = i256.Add(i256.Rsh(i256.Mul(q, value), 96), q5)

	// # The polynomial `q` has no zeros in the range because all its roots are complex.
	// # No scaling is required, as `p` is already "2 ** 96" too large. Also,
	// # `r` is in the range "(0.09, 0.25) * 2**96" after the division.
	r := i256.Div(p, q)

	// # To finalise the calculation, we have to multiply `r` by:
	// #   - the scale factor "s = ~6.031367120",
	// #   - the factor "2 ** k" from the range reduction, and
	// #   - the factor "1e18 / 2 ** 96" for the base conversion.
	// # We do this all at once, with an intermediate result in "2**213" base,
	// # so that the final right shift always gives a positive value.

	// # Note that to circumvent Vyper's safecast feature for the potentially
	// # negative parameter value `r`, we first convert `r` to `bytes32` and
	// # subsequently to `uint256`. Remember that the EVM default behaviour is
	// # to use two's complement representation to handle signed integers.
	tmp := number.Mul(i256.UnsafeToUInt256(r), scaleFactor)
	n := 195 - k.Int64()
	return new(uint256.Int).Rsh(tmp, uint(n)), nil
}

func FullMulDiv(a, b, c *uint256.Int) (*uint256.Int, error) {
	var result uint256.Int
	_, overflow := result.MulDivOverflow(a, b, c)
	if overflow {
		return nil, ErrOverflow
	}
	return &result, nil
}

func RoundUpFullMulDivResult(a, b, c, estimate *uint256.Int) (*uint256.Int, error) {
	var result uint256.Int
	result.Set(estimate)

	var remainder uint256.Int
	remainder.Mul(a, b)
	remainder.Mod(&remainder, c)

	if !remainder.IsZero() {
		result.AddUint64(&result, 1)

		if result.IsZero() {
			return nil, errors.New("FullMulDivFailed")
		}
	}

	return &result, nil
}

func FullMulDivUp(a, b, c *uint256.Int) (*uint256.Int, error) {
	if c.IsZero() {
		return nil, ErrOverflow
	}

	var product uint256.Int
	product.Mul(a, b)

	var remainder uint256.Int
	remainder.Mod(&product, c)

	var result uint256.Int
	result.Div(&product, c)

	if !remainder.IsZero() {
		result.Add(&result, uint256.NewInt(1))
	}

	return &result, nil
}

// FullMulX96Up performs full multiplication with X96 scaling and rounds up
func FullMulX96Up(a, b *uint256.Int) (*uint256.Int, error) {
	var product uint256.Int
	product.Mul(a, b)

	var result uint256.Int
	result.Div(&product, Q96)

	var remainder uint256.Int
	remainder.Mod(&product, Q96)

	if !remainder.IsZero() {
		result.Add(&result, uint256.NewInt(1))
	}

	return &result, nil
}

func FullMulX96(a, b *uint256.Int) (*uint256.Int, error) {
	var result uint256.Int
	_, overflow := result.MulOverflow(a, b)
	if overflow {
		return nil, errors.New("FullMulX96Overflow")
	}

	result.Rsh(&result, 96)
	return &result, nil
}

func FromIdleBalance(idleBalance [32]byte) (*uint256.Int, bool) {
	var balance uint256.Int
	balance.SetBytes(idleBalance[:])

	// Clear the highest bit to get the raw balance
	// The highest bit is used to indicate which token (0 or 1)
	var highestBit uint256.Int
	highestBit.SetUint64(1)
	highestBit.Lsh(&highestBit, 255)

	// Clear the highest bit by using AND with inverted mask
	var mask uint256.Int
	mask.SetAllOne()
	mask.Xor(&mask, &highestBit)
	balance.And(&balance, &mask)

	// Check highest bit for isToken0
	isToken0 := (idleBalance[0] & 0x80) == 0

	return &balance, isToken0
}

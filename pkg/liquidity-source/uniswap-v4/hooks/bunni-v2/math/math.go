package math

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/int256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

func MulDiv(a, b, denominator *uint256.Int) *uint256.Int {
	res, _ := v3Utils.MulDiv(a, b, denominator)
	return res
}

func MulDivUp(a, b, denominator *uint256.Int) *uint256.Int {
	res, _ := v3Utils.MulDivRoundingUp(a, b, denominator)
	return res
}

func DivUp(a, b *uint256.Int) *uint256.Int {
	var res uint256.Int
	v3Utils.DivRoundingUp(a, b, &res)
	return &res
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

func lnQ96(x *int256.Int) (*int256.Int, int, error) {
	if x.Sign() <= 0 {
		return nil, 0, errors.New("LnQ96Undefined")
	}

	ux := i256.UnsafeToUInt256(x)
	msb := ux.BitLen() - 1
	k := msb - 96

	var value *int256.Int
	if k > 0 {
		value = i256.Rsh(x, uint(k))
	} else if k < 0 {
		value = i256.Lsh(x, uint(-k))
	} else {
		value = x
	}

	p := i256.Sub(
		i256.Rsh(
			i256.Mul(
				i256.Add(lnQ96A0,
					i256.Rsh(
						i256.Mul(
							i256.Add(lnQ96A1,
								i256.Rsh(
									i256.Mul(i256.Add(lnQ96A2, value), value),
									96,
								),
							),
							value,
						),
						96,
					),
				),
				value,
			),
			96,
		),
		lnQ96B0,
	)
	p = i256.Sub(i256.Rsh(i256.Mul(p, value), 96), lnQ96B1)
	p = i256.Sub(i256.Rsh(i256.Mul(p, value), 96), lnQ96B2)
	p = i256.Sub(i256.Mul(p, value), i256.Lsh(lnQ96C, 96))

	q := i256.Add(lnQ96Q0, value)
	q = i256.Add(lnQ96Q1, i256.Rsh(i256.Mul(value, q), 96))
	q = i256.Add(lnQ96Q2, i256.Rsh(i256.Mul(value, q), 96))
	q = i256.Add(lnQ96Q3, i256.Rsh(i256.Mul(value, q), 96))
	q = i256.Add(lnQ96Q4, i256.Rsh(i256.Mul(value, q), 96))
	q = i256.Add(lnQ96Q5, i256.Rsh(i256.Mul(value, q), 96))
	q = i256.Add(lnQ96Q6, i256.Rsh(i256.Mul(value, q), 96))

	p = i256.Div(p, q)

	p = i256.Mul(lnQ96Scale, p)

	if k != 0 {
		p = i256.Add(p, i256.Mul(lnQ96Ln2Scaled2Pow192, int256.NewInt(int64(k))))
	}

	return p, k, nil
}

func LnQ96(x *int256.Int) (*int256.Int, error) {
	p, _, err := lnQ96(x)
	if err != nil {
		return nil, err
	}
	return i256.Rsh(p, 96), nil
}

func LnQ96RoundingUp(x *int256.Int) (*int256.Int, error) {
	p, _, err := lnQ96(x)
	if err != nil {
		return nil, err
	}

	res := i256.Rsh(p, 96)
	if i256.Lsh(res, 96).Cmp(p) != 0 {
		res.Add(res, i256.Number_1)
	}
	return res, nil
}

func SDivWad(x, y *int256.Int) (*int256.Int, error) {
	// Equivalent to require(y != 0 && ((x * WAD) / WAD == x))
	if y.Sign() == 0 {
		return nil, errors.New("SDivWadFailed")
	}

	var z int256.Int
	z.Mul(x, WAD_INT)

	if new(int256.Int).Quo(&z, WAD_INT).Cmp(x) != 0 {
		return nil, errors.New("SDivWadFailed")
	}

	z.Quo(&z, y)

	return &z, nil
}

// XWadToRoundedTick converts xWad to rounded tick (placeholder)
func XWadToRoundedTick(xWad *int256.Int, mu int, tickSpacing int, roundUp bool) int {
	x := int(new(int256.Int).Quo(xWad, WAD_INT).Int64())

	tmp := new(int256.Int).Rem(xWad, WAD_INT)
	if roundUp {
		if xWad.Sign() > 0 && tmp.Sign() != 0 {
			x++
		}
	} else {
		if xWad.Sign() < 0 && tmp.Sign() != 0 {
			x--
		}
	}

	return x*tickSpacing + mu
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

func RoundUpFullMulDivResult(x, y, d, resultRoundedDown *uint256.Int) (*uint256.Int, error) {
	var remainder uint256.Int
	remainder.MulMod(x, y, d)
	if remainder.IsZero() {
		return resultRoundedDown, nil
	}
	var res uint256.Int
	res.Set(resultRoundedDown)
	res.AddUint64(&res, 1)
	if res.IsZero() {
		return nil, ErrOverflow
	}
	return &res, nil
}

func FromIdleBalance(idleBalance [32]byte) (*uint256.Int, bool) {
	isToken0 := (idleBalance[0] & 0x80) != 0

	var raw [32]byte
	copy(raw[:], idleBalance[:])
	raw[0] &^= 0x80

	var balance uint256.Int
	balance.SetBytes(raw[:])
	return &balance, isToken0
}

func ToIdleBalance(rawBalance *uint256.Int, isToken0 bool) ([32]byte, error) {
	if rawBalance.Gt(_BALANCE_MASK) {
		return [32]byte{}, errors.New("IdleBalanceLibrary__BalanceOverflow")
	}

	var idleBalance [32]byte
	rawBalance.WriteToSlice(idleBalance[:])

	if isToken0 {
		idleBalance[0] |= 0x80
	}

	return idleBalance, nil
}

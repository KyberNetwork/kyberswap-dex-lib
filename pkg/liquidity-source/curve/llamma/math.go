package llamma

import (
	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

func maxUint256(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) > 0 {
		return a
	}
	return b
}

func minUint256(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) < 0 {
		return a
	}
	return b
}

func wadExp(x *int256.Int) (*uint256.Int, error) {
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
	if x.Cmp(i256.MustFromDecimal("-42139678854452767551")) <= 0 {
		return uint256.NewInt(0), nil
	}

	// # When the result is "> (2 ** 255 - 1) / 1e18" we cannot represent it as a signed integer.
	// # This happens when "x >= floor(log((2 ** 255 - 1) / 1e18) * 1e18) ~ 135".
	if x.Cmp(i256.MustFromDecimal("135305999368893231589")) >= 0 {
		return nil, ErrWadExpOverflow
	}

	// # `x` is now in the range "(-42, 136) * 1e18". Convert to "(-42, 136) * 2 ** 96" for higher
	// # intermediate precision and a binary base. This base conversion is a multiplication with
	// # "1e18 / 2 ** 96 = 5 ** 18 / 2 ** 78".
	value := i256.Div(i256.Lsh(x, 78), i256.MustFromDecimal("3814697265625"))

	// # Reduce the range of `x` to "(-½ ln 2, ½ ln 2) * 2 ** 96" by factoring out powers of two
	// # so that "exp(x) = exp(x') * 2 ** k", where `k` is a signer integer. Solving this gives
	// # "k = round(x / log(2))" and "x' = x - k * log(2)". Thus, `k` is in the range "[-61, 195]".
	k := i256.Rsh(
		i256.Add(
			i256.Div(
				i256.Lsh(value, 96),
				i256.MustFromDecimal("54916777467707473351141471128")),
			i256.MustFromDecimal("39614081257132168796771975168")),
		96)
	value = i256.Sub(value, i256.Mul(k, i256.MustFromDecimal("54916777467707473351141471128")))

	// # Evaluate using a "(6, 7)"-term rational approximation. Since `p` is monic,
	// # we will multiply by a scaling factor later.
	y := i256.Add(
		i256.Rsh(
			i256.Mul(i256.Add(value, i256.MustFromDecimal("1346386616545796478920950773328")), value),
			96),
		i256.MustFromDecimal("57155421227552351082224309758442"))
	p := i256.Add(
		i256.Mul(
			i256.Add(
				i256.Rsh(
					i256.Mul(
						i256.Sub(i256.Add(y, value),
							i256.MustFromDecimal("94201549194550492254356042504812")),
						y),
					96),
				i256.MustFromDecimal("28719021644029726153956944680412240")),
			value),
		i256.Lsh(i256.MustFromDecimal("4385272521454847904659076985693276"), 96),
	)

	// # We leave `p` in the "2 ** 192" base so that we do not have to scale it up
	// # again for the division.
	q := i256.Add(
		i256.Rsh(
			i256.Mul(i256.Sub(value, i256.MustFromDecimal("2855989394907223263936484059900")), value),
			96),
		i256.MustFromDecimal("50020603652535783019961831881945"))
	q = i256.Sub(i256.Rsh(i256.Mul(q, value), 96), i256.MustFromDecimal("533845033583426703283633433725380"))
	q = i256.Add(i256.Rsh(i256.Mul(q, value), 96), i256.MustFromDecimal("3604857256930695427073651918091429"))
	q = i256.Sub(i256.Rsh(i256.Mul(q, value), 96), i256.MustFromDecimal("14423608567350463180887372962807573"))
	q = i256.Add(i256.Rsh(i256.Mul(q, value), 96), i256.MustFromDecimal("26449188498355588339934803723976023"))

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
	tmp := number.Mul(i256.UnsafeToUInt256(r), uint256.MustFromDecimal("3822833074963236453042738258902158003155416615667"))
	n := 195 - k.Int64()
	return new(uint256.Int).Rsh(tmp, uint(n)), nil
}

func lnInt(x *uint256.Int) *int256.Int {
	var res uint256.Int
	for i := 0; i < 8; i++ {
		t := new(uint256.Int).Exp(number.Number_2, uint256.NewInt(uint64(7-i)))
		p := new(uint256.Int).Exp(number.Number_2, t)
		if x.Cmp(new(uint256.Int).Mul(p, number.Number_1e18)) >= 0 {
			x.Div(x, p)
			res.Add(&res, new(uint256.Int).Mul(t, number.Number_1e18))
		}
	}

	d := new(uint256.Int).Set(number.Number_1e18)
	for i := 0; i < 59; i++ {
		if x.Cmp(new(uint256.Int).Mul(number.Number_2, number.Number_1e18)) >= 0 {
			res.Add(&res, d)
			x.Div(x, number.Number_2)
		}
		x.Mul(x, x).Div(x, number.Number_1e18)
		d.Div(d, number.Number_2)
	}

	// Now res = log2(x)
	// ln(x) = log2(x) / log2(e)
	res.Mul(&res, number.Number_1e18).Div(&res, uint256.NewInt(1442695040888963328))

	return i256.SafeToInt256(&res)
}

package math

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var (
	UNIT_LPOTD   = uint256.NewInt(262144)
	UNIT         = uint256.NewInt(1e18)
	UNIT_INVERSE = uint256.MustFromDecimal("78156646155174841979727994598816262306175212592076161876661508869554232690281")
)

var (
	Err_PRBMath_MulDiv18_Overflow = errors.New("PRBMath_MulDiv18_Overflow")
	Err_PRBMath_MulDiv_Overflow   = errors.New("PRBMath_MulDiv_Overflow")

	ErrDivideByZero = errors.New("divide by zero")
	ErrOverflow     = errors.New("overflow")
)

var Common *common

type common struct{}

// https://github.com/velocore/velocore-contracts/blob/master/lib/prb-math/src/Common.sol#L495
func (*common) MulDiv18(x, y *uint256.Int) (*uint256.Int, error) {
	var prod0, prod1 *uint256.Int
	{
		mm := ASM.MulMod(x, y, ASM.Not(number.Zero))
		prod0 = ASM.Mul(x, y)
		prod1 = ASM.Sub(
			ASM.Sub(mm, prod0),
			ASM.Lt(mm, prod0),
		)
	}

	if prod1.IsZero() {
		return new(uint256.Int).Div(prod0, UNIT), nil
	}

	if prod1.Cmp(UNIT) >= 0 {
		return nil, Err_PRBMath_MulDiv18_Overflow
	}

	var remainder, result *uint256.Int
	{
		remainder = ASM.MulMod(x, y, UNIT)
		result = ASM.Mul(
			ASM.Or(
				ASM.Div(ASM.Sub(prod0, remainder), UNIT_LPOTD),
				ASM.Mul(ASM.Sub(prod1, ASM.Gt(remainder, prod0)), ASM.Add(ASM.Div(ASM.Sub(number.Zero, UNIT_LPOTD), UNIT_LPOTD), number.Number_1)),
			),
			UNIT_INVERSE,
		)
	}

	return result, nil
}

// https://github.com/velocore/velocore-contracts/blob/master/lib/prb-math/src/Common.sol#L387
func (*common) MulDiv(x, y, denominator *uint256.Int) (*uint256.Int, error) {
	var prod0, prod1 *uint256.Int
	{
		mm := ASM.MulMod(x, y, ASM.Not(number.Zero))
		prod0 = ASM.Mul(x, y)
		prod1 = ASM.Sub(ASM.Sub(mm, prod0), ASM.Lt(mm, prod0))
	}

	if prod1.IsZero() {
		return new(uint256.Int).Div(prod0, denominator), nil
	}

	if prod1.Cmp(denominator) >= 0 {
		return nil, Err_PRBMath_MulDiv_Overflow
	}

	var remainder *uint256.Int
	{

		remainder = ASM.MulMod(x, y, denominator)

		prod1 = ASM.Sub(prod1, ASM.Gt(remainder, prod0))
		prod0 = ASM.Sub(prod0, remainder)
	}

	lpotdod := new(uint256.Int).And(
		denominator,
		new(uint256.Int).Add(
			new(uint256.Int).Not(denominator),
			number.Number_1,
		),
	)

	var flippedLpotdod *uint256.Int
	{
		denominator = ASM.Div(denominator, lpotdod)

		prod0 = ASM.Div(prod0, lpotdod)

		flippedLpotdod = ASM.Add(
			ASM.Div(
				ASM.Sub(number.Zero, lpotdod),
				lpotdod,
			),
			number.Number_1,
		)
	}

	prod0 = new(uint256.Int).Or(
		prod0,
		new(uint256.Int).Mul(prod1, flippedLpotdod),
	)

	inverse := new(uint256.Int).Exp(new(uint256.Int).Mul(number.Number_3, denominator), number.Number_2)
	for i := 0; i < 6; i++ {
		inverse = new(uint256.Int).Mul(
			inverse,
			new(uint256.Int).Sub(
				number.Number_2,
				new(uint256.Int).Mul(
					denominator,
					inverse,
				),
			),
		)
	}

	result := new(uint256.Int).Mul(prod0, inverse)

	return result, nil
}

// https://github.com/velocore/velocore-contracts/blob/master/lib/prb-math/src/Common.sol#L54
func (*common) Exp2(x *uint256.Int) *uint256.Int {
	result := uint256.MustFromHex("0x800000000000000000000000000000000000000000000000")

	type pair struct {
		X string
		Y string
	}
	const n = 8
	v := [n]string{
		"FF00000000000000",
		"FF000000000000",
		"FF0000000000",
		"FF00000000",
		"FF000000",
		"FF0000",
		"FF00",
		"FF",
	}
	pairs := [n][n]pair{
		{
			{"0x8000000000000000", "0x16A09E667F3BCC909"},
			{"0x4000000000000000", "0x1306FE0A31B7152DF"},
			{"0x2000000000000000", "0x1172B83C7D517ADCE"},
			{"0x1000000000000000", "0x10B5586CF9890F62A"},
			{"0x800000000000000", "0x1059B0D31585743AE"},
			{"0x400000000000000", "0x102C9A3E778060EE7"},
			{"0x200000000000000", "0x10163DA9FB33356D8"},
			{"0x100000000000000", "0x100B1AFA5ABCBED61"},
		},
		{
			{"0x80000000000000", "0x10058C86DA1C09EA2"},
			{"0x40000000000000", "0x1002C605E2E8CEC50"},
			{"0x20000000000000", "0x100162F3904051FA1"},
			{"0x10000000000000", "0x1000B175EFFDC76BA"},
			{"0x8000000000000", "0x100058BA01FB9F96D"},
			{"0x4000000000000", "0x10002C5CC37DA9492"},
			{"0x2000000000000", "0x1000162E525EE0547"},
			{"0x1000000000000", "0x10000B17255775C04"},
		},
		{
			{"0x800000000000", "0x1000058B91B5BC9AE"},
			{"0x400000000000", "0x100002C5C89D5EC6D"},
			{"0x200000000000", "0x10000162E43F4F831"},
			{"0x100000000000", "0x100000B1721BCFC9A"},
			{"0x80000000000", "0x10000058B90CF1E6E"},
			{"0x40000000000", "0x1000002C5C863B73F"},
			{"0x20000000000", "0x100000162E430E5A2"},
			{"0x10000000000", "0x1000000B172183551"},
		},
		{
			{"0x8000000000", "0x100000058B90C0B49"},
			{"0x4000000000", "0x10000002C5C8601CC"},
			{"0x2000000000", "0x1000000162E42FFF0"},
			{"0x1000000000", "0x10000000B17217FBB"},
			{"0x800000000", "0x1000000058B90BFCE"},
			{"0x400000000", "0x100000002C5C85FE3"},
			{"0x200000000", "0x10000000162E42FF1"},
			{"0x100000000", "0x100000000B17217F8"},
		},
		{
			{"0x80000000", "0x10000000058B90BFC"},
			{"0x40000000", "0x1000000002C5C85FE"},
			{"0x20000000", "0x100000000162E42FF"},
			{"0x10000000", "0x1000000000B17217F"},
			{"0x8000000", "0x100000000058B90C0"},
			{"0x4000000", "0x10000000002C5C860"},
			{"0x2000000", "0x1000000000162E430"},
			{"0x1000000", "0x10000000000B17218"},
		},
		{
			{"0x800000", "0x1000000000058B90C"},
			{"0x400000", "0x100000000002C5C86"},
			{"0x200000", "0x10000000000162E43"},
			{"0x100000", "0x100000000000B1721"},
			{"0x80000", "0x10000000000058B91"},
			{"0x40000", "0x1000000000002C5C8"},
			{"0x20000", "0x100000000000162E4"},
			{"0x10000", "0x1000000000000B172"},
		},
		{
			{"0x8000", "0x100000000000058B9"},
			{"0x4000", "0x10000000000002C5D"},
			{"0x2000", "0x1000000000000162E"},
			{"0x1000", "0x10000000000000B17"},
			{"0x800", "0x1000000000000058C"},
			{"0x400", "0x100000000000002C6"},
			{"0x200", "0x10000000000000163"},
			{"0x100", "0x100000000000000B1"},
		},
		{
			{"0x80", "0x10000000000000059"},
			{"0x40", "0x1000000000000002C"},
			{"0x20", "0x10000000000000016"},
			{"0x10", "0x1000000000000000B"},
			{"0x8", "0x10000000000000006"},
			{"0x4", "0x10000000000000003"},
			{"0x2", "0x10000000000000001"},
			{"0x1", "0x10000000000000001"},
		},
	}

	for i := 0; i < n; i++ {
		vi := uint256.MustFromHex(v[i])
		if !new(uint256.Int).And(x, vi).Gt(number.Zero) {
			continue
		}

		for j := 0; j < n; j++ {
			xj := uint256.MustFromHex(pairs[i][j].X)
			if !new(uint256.Int).And(x, xj).Gt(number.Zero) {
				continue
			}

			yj := uint256.MustFromHex(pairs[i][j].Y)
			result = new(uint256.Int).Rsh(new(uint256.Int).Mul(result, yj), 64)
		}
	}

	result = new(uint256.Int).Mul(result, UNIT)
	result = new(uint256.Int).Rsh(
		result,
		uint(new(uint256.Int).Sub(
			uint256.NewInt(191),
			new(uint256.Int).Rsh(x, 64),
		).Uint64()),
	)

	return result
}

// https://github.com/velocore/velocore-contracts/blob/master/lib/prb-math/src/Common.sol#L320
//
// NOTE: Msb is modified from the original implementation.
func (*common) Msb(x *uint256.Int) *uint256.Int {
	l, r := 0, 256
	for r-l > 1 {
		m := (l + r) >> 1
		p := new(uint256.Int).Lsh(number.Number_1, uint(m))
		if p.Cmp(x) <= 0 {
			l = m
		} else {
			r = m
		}
	}
	return uint256.NewInt(uint64(l))
}

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/src/pools/constant-product/ConstantProductPool.sol#L29
func (*common) CeilDivUnsafe(x, y *uint256.Int) *uint256.Int {
	return new(uint256.Int).Div(
		new(uint256.Int).Sub(
			new(uint256.Int).Add(x, y),
			number.Number_1,
		), y,
	)
}

// https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/utils/math/Math.sol#L107
func (*common) CeilDiv(x, y *uint256.Int) (*uint256.Int, error) {
	if y.IsZero() {
		return nil, ErrDivideByZero
	}

	if x.IsZero() {
		return number.Zero, nil
	}

	return new(uint256.Int).Add(
		new(uint256.Int).Div(
			new(uint256.Int).Sub(x, number.Number_1),
			y,
		),
		number.Number_1,
	), nil
}

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/src/lib/RPow.sol#L22
func (*common) RPow(x, n, base *uint256.Int) (*uint256.Int, error) {
	if x.IsZero() {
		if n.IsZero() {
			return new(uint256.Int).Set(base), nil
		}

		return number.Zero, nil
	}

	z := x
	if ASM.Mod(n, number.Number_2).IsZero() {
		z = base
	}

	half := ASM.Div(base, number.Number_2)
	for i := ASM.Div(n, number.Number_2); i.Gt(number.Zero); i = ASM.Div(i, number.Number_2) {
		xx := ASM.Mul(x, x)

		if !ASM.Div(xx, x).Eq(x) {
			return nil, ErrOverflow
		}

		xxRound := ASM.Add(xx, half)
		if xxRound.Lt(xx) {
			return nil, ErrOverflow
		}

		x := ASM.Div(xxRound, base)
		if !ASM.Mod(i, number.Number_2).IsZero() {
			zx := ASM.Mul(z, x)

			if !x.IsZero() && !ASM.Div(zx, x).Eq(z) {
				return nil, ErrOverflow
			}

			zxRound := ASM.Add(zx, half)
			if zxRound.Lt(zx) {
				return nil, ErrOverflow
			}

			z = ASM.Div(zxRound, base)
		}
	}

	return z, nil
}

package sd59x18

import "math/big"

// https://github.com/PaulRBerg/prb-math/blob/main/src/Common.sol

func mulDiv18(x *big.Int, y *big.Int) (*big.Int, error) {
	var (
		prod0 *big.Int
		prod1 *big.Int
	)

	not0 := new(big.Int).Sub(
		new(big.Int).Exp(bigint2, bigint256, nil),
		bigint1,
	)

	mm := new(big.Int).Mod(
		new(big.Int).Mul(x, y),
		not0,
	)
	prod0 = new(big.Int).Mul(x, y)
	prod1 = new(big.Int).Sub(
		new(big.Int).Sub(mm, prod0),
		lt(mm, prod0),
	)

	if prod1.Cmp(bigint0) == 0 {
		return new(big.Int).Div(prod0, unit), nil
	}

	if prod1.Cmp(unit) >= 0 {
		return nil, ErrMathMulDiv18Overflow
	}

	remainder := new(big.Int).Mod(
		new(big.Int).Mul(x, y),
		unit,
	)
	result := new(big.Int).Mul(
		new(big.Int).Or(
			new(big.Int).Div(
				new(big.Int).Sub(prod0, remainder),
				unitLPOTD,
			),
			new(big.Int).Mul(
				new(big.Int).Sub(prod1, gt(remainder, prod0)),
				new(big.Int).Add(
					new(big.Int).Div(
						new(big.Int).Sub(bigint0, unitLPOTD),
						unitLPOTD,
					),
					bigint1,
				),
			),
		),
		unitInverse,
	)

	return result, nil
}

func mulDiv(x *big.Int, y *big.Int, denominator *big.Int) (*big.Int, error) {
	var (
		prod0 *big.Int
		prod1 *big.Int
	)

	not0 := new(big.Int).Sub(
		new(big.Int).Exp(bigint2, bigint256, nil),
		bigint1,
	)
	mm := new(big.Int).Mod(
		new(big.Int).Mul(x, y),
		not0,
	)
	prod0 = new(big.Int).Mul(x, y)
	prod1 = new(big.Int).Sub(
		new(big.Int).Sub(mm, prod0),
		lt(mm, prod0),
	)

	if prod1.Cmp(bigint0) == 0 {
		return new(big.Int).Div(prod0, denominator), nil
	}

	if prod1.Cmp(denominator) >= 0 {
		return nil, ErrMathMulDivOverflow
	}

	remainder := new(big.Int).Mod(
		new(big.Int).Mul(x, y),
		denominator,
	)
	prod1 = new(big.Int).Sub(prod1, gt(remainder, prod0))
	prod0 = new(big.Int).Sub(prod0, remainder)

	lpotdod := new(big.Int).And(
		denominator,
		new(big.Int).Add(
			new(big.Int).Not(denominator),
			bigint1,
		),
	)

	denominator = new(big.Int).Div(denominator, lpotdod)
	prod0 = new(big.Int).Div(prod0, lpotdod)

	flippedLpotdod := new(big.Int).Add(
		new(big.Int).Div(
			new(big.Int).Sub(bigint0, lpotdod),
			lpotdod,
		),
		bigint1,
	)

	prod0 = new(big.Int).Or(
		prod0,
		new(big.Int).Mul(
			prod1,
			flippedLpotdod,
		),
	)

	inverse := new(big.Int).Mul(
		new(big.Int).Mul(denominator, denominator),
		big.NewInt(9),
	)

	for i := 0; i < 6; i++ {
		inverse = new(big.Int).Mul(
			inverse,
			new(big.Int).Sub(
				bigint2,
				new(big.Int).Mul(
					denominator,
					inverse,
				),
			),
		)
	}

	result := new(big.Int).Mul(prod0, inverse)
	return result, nil
}

func lt(x *big.Int, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return bigint1
	}
	return bigint0
}

func gt(x *big.Int, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return bigint1
	}
	return bigint0
}

func msb(x *big.Int) *big.Int {
	l := 0
	r := 256
	two := bigint2
	for r-l > 1 {
		m := (l + r) >> 1
		twoPowM := new(big.Int).Exp(two, big.NewInt(int64(m)), nil)
		if twoPowM.Cmp(x) <= 0 {
			l = m
		} else {
			r = m
		}
	}
	return big.NewInt(int64(l))
}

func exp2(x *big.Int) *big.Int {
	// Start from 0.5 in the 192.64-bit fixed-point format.
	result, _ := new(big.Int).SetString("800000000000000000000000000000000000000000000000", 16)
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
	type pair struct {
		X string
		Y string
	}
	pairs := [n][n]pair{
		{
			{"8000000000000000", "16A09E667F3BCC909"},
			{"4000000000000000", "1306FE0A31B7152DF"},
			{"2000000000000000", "1172B83C7D517ADCE"},
			{"1000000000000000", "10B5586CF9890F62A"},
			{"800000000000000", "1059B0D31585743AE"},
			{"400000000000000", "102C9A3E778060EE7"},
			{"200000000000000", "10163DA9FB33356D8"},
			{"100000000000000", "100B1AFA5ABCBED61"},
		},
		{
			{"80000000000000", "10058C86DA1C09EA2"},
			{"40000000000000", "1002C605E2E8CEC50"},
			{"20000000000000", "100162F3904051FA1"},
			{"10000000000000", "1000B175EFFDC76BA"},
			{"8000000000000", "100058BA01FB9F96D"},
			{"4000000000000", "10002C5CC37DA9492"},
			{"2000000000000", "1000162E525EE0547"},
			{"1000000000000", "10000B17255775C04"},
		},
		{
			{"800000000000", "1000058B91B5BC9AE"},
			{"400000000000", "100002C5C89D5EC6D"},
			{"200000000000", "10000162E43F4F831"},
			{"100000000000", "100000B1721BCFC9A"},
			{"80000000000", "10000058B90CF1E6E"},
			{"40000000000", "1000002C5C863B73F"},
			{"20000000000", "100000162E430E5A2"},
			{"10000000000", "1000000B172183551"},
		},
		{
			{"8000000000", "100000058B90C0B49"},
			{"4000000000", "10000002C5C8601CC"},
			{"2000000000", "1000000162E42FFF0"},
			{"1000000000", "10000000B17217FBB"},
			{"800000000", "1000000058B90BFCE"},
			{"400000000", "100000002C5C85FE3"},
			{"200000000", "10000000162E42FF1"},
			{"100000000", "100000000B17217F8"},
		},
		{
			{"80000000", "10000000058B90BFC"},
			{"40000000", "1000000002C5C85FE"},
			{"20000000", "100000000162E42FF"},
			{"10000000", "1000000000B17217F"},
			{"8000000", "100000000058B90C0"},
			{"4000000", "10000000002C5C860"},
			{"2000000", "1000000000162E430"},
			{"1000000", "10000000000B17218"},
		},
		{
			{"800000", "1000000000058B90C"},
			{"400000", "100000000002C5C86"},
			{"200000", "10000000000162E43"},
			{"100000", "100000000000B1721"},
			{"80000", "10000000000058B91"},
			{"40000", "1000000000002C5C8"},
			{"20000", "100000000000162E4"},
			{"10000", "1000000000000B172"},
		},
		{
			{"8000", "100000000000058B9"},
			{"4000", "10000000000002C5D"},
			{"2000", "1000000000000162E"},
			{"1000", "10000000000000B17"},
			{"800", "1000000000000058C"},
			{"400", "100000000000002C6"},
			{"200", "10000000000000163"},
			{"100", "100000000000000B1"},
		},
		{
			{"80", "10000000000000059"},
			{"40", "1000000000000002C"},
			{"20", "10000000000000016"},
			{"10", "1000000000000000B"},
			{"8", "10000000000000006"},
			{"4", "10000000000000003"},
			{"2", "10000000000000001"},
			{"1", "10000000000000001"},
		},
	}

	for i := 0; i < n; i++ {
		vi, _ := new(big.Int).SetString(v[i], 16)
		if new(big.Int).And(x, vi).Cmp(bigint0) <= 0 {
			continue
		}

		for j := 0; j < n; j++ {
			xj, _ := new(big.Int).SetString(pairs[i][j].X, 16)
			if new(big.Int).And(x, xj).Cmp(bigint0) <= 0 {
				continue
			}

			yj, _ := new(big.Int).SetString(pairs[i][j].Y, 16)
			result = new(big.Int).Rsh(new(big.Int).Mul(result, yj), 64)
		}
	}

	result = new(big.Int).Mul(result, unit)
	result = new(big.Int).Rsh(
		result,
		uint(new(big.Int).Sub(
			big.NewInt(191),
			new(big.Int).Rsh(x, 64),
		).Uint64()),
	)

	return result
}

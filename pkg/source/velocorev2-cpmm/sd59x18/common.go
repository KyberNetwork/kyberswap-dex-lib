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
		return new(big.Int).Div(prod0, uUnit), nil
	}

	if prod1.Cmp(uUnit) >= 0 {
		return nil, ErrMathMulDiv18Overflow
	}

	remainder := new(big.Int).Mod(
		new(big.Int).Mul(x, y),
		uUnit,
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
		return big.NewInt(1)
	}
	return big.NewInt(0)
}

func gt(x *big.Int, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return big.NewInt(1)
	}
	return big.NewInt(0)
}

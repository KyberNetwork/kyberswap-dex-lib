package liquiditybookv20

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
)

func getPriceFromID(id uint32, binStep uint16) (*big.Int, error) {
	base := getBase(binStep)
	exponent := getExponent(id)
	return pow(base, exponent)
}

func getBase(binStep uint16) *big.Int {
	u := new(big.Int).Lsh(big.NewInt(int64(binStep)), scaleOffset)
	return new(big.Int).Add(scale, new(big.Int).Div(u, big.NewInt(basisPointMax)))
}

func getExponent(id uint32) *big.Int {
	return new(big.Int).Sub(big.NewInt(int64(id)), big.NewInt(realIDShift))
}

func pow(x *big.Int, y *big.Int) (*big.Int, error) {
	var (
		invert bool
		absY   *big.Int
		result = big.NewInt(0)
	)

	if y.Cmp(integer.Zero()) == 0 {
		return scale, nil
	}

	absY = new(big.Int).Abs(y)
	if y.Sign() < 0 {
		invert = !invert
	}

	u, _ := new(big.Int).SetString("100000", 16)
	if absY.Cmp(u) < 0 {
		result = scale

		squared := x
		v, _ := new(big.Int).SetString("ffffffffffffffffffffffffffffffff", 16)
		if x.Cmp(v) > 0 {
			not0 := maxUint256
			squared = new(big.Int).Div(not0, squared)

			invert = !invert
		}

		for i := 0x1; i <= 0x80000; i <<= 1 {
			and := new(big.Int).And(absY, big.NewInt(int64(i)))
			if and.Cmp(integer.Zero()) != 0 {
				result = new(big.Int).Rsh(
					new(big.Int).Mul(result, squared),
					128,
				)
			}
			if i < 0x80000 {
				squared = new(big.Int).Rsh(
					new(big.Int).Mul(squared, squared),
					128,
				)
			}
		}
	}

	if result.Cmp(integer.Zero()) == 0 {
		return nil, ErrPowUnderflow
	}

	if invert {
		v := new(big.Int).Sub(new(big.Int).Lsh(integer.One(), 256), integer.One())
		result = new(big.Int).Div(v, result)
	}

	return result, nil
}

func shiftDivRoundUp(x *big.Int, offset uint8, denominator *big.Int) (*big.Int, error) {
	result, err := shiftDivRoundDown(x, offset, denominator)
	if err != nil {
		return nil, err
	}

	if denominator.Cmp(integer.Zero()) == 0 {
		return integer.Zero(), nil
	}
	v := new(big.Int).Mod(
		new(big.Int).Mul(x, new(big.Int).Lsh(integer.One(), uint(offset))),
		denominator,
	)
	if v.Cmp(integer.Zero()) != 0 {
		result = new(big.Int).Add(result, integer.One())
	}

	return result, nil
}

func shiftDivRoundDown(x *big.Int, offset uint8, denominator *big.Int) (*big.Int, error) {
	var (
		prod0, prod1 *big.Int
	)

	prod0 = new(big.Int).Lsh(x, uint(offset))
	prod1 = new(big.Int).Rsh(x, uint(256-int(offset)))

	y := new(big.Int).Lsh(integer.One(), uint(offset))

	return getEndOfDivRoundDown(x, y, denominator, prod0, prod1)
}

func getEndOfDivRoundDown(
	x *big.Int,
	y *big.Int,
	denominator *big.Int,
	prod0 *big.Int,
	prod1 *big.Int,
) (*big.Int, error) {
	if prod1.Cmp(integer.Zero()) == 0 {
		return new(big.Int).Div(prod0, denominator), nil
	}

	if prod1.Cmp(denominator) >= 0 {
		return nil, ErrMulDivOverflow
	}

	var remainder *big.Int
	if denominator.Cmp(integer.Zero()) == 0 {
		remainder = integer.Zero()
	} else {
		remainder = new(big.Int).Mod(new(big.Int).Mul(x, y), denominator)
	}

	prod1 = new(big.Int).Sub(prod1, gt(remainder, prod0))
	prod0 = new(big.Int).Sub(prod0, remainder)

	// bitwiseNotDenominator = ~denominator, denominator has type uint256
	lpotdod := new(big.Int).And(denominator, new(big.Int).Add(bitwiseNotUint256(denominator), integer.One()))

	denominator = new(big.Int).Div(denominator, lpotdod)

	prod0 = new(big.Int).Div(prod0, lpotdod)

	lpotdod = new(big.Int).Add(
		new(big.Int).Div(new(big.Int).Sub(integer.Zero(), lpotdod), lpotdod),
		integer.One(),
	)

	prod0 = new(big.Int).Or(prod0, new(big.Int).Mul(prod1, lpotdod))

	inverse := new(big.Int).Mul(new(big.Int).Mul(denominator, denominator), big.NewInt(9))

	for i := 0; i < 6; i++ {
		inverse = new(big.Int).Mul(
			inverse,
			new(big.Int).Sub(integer.Two(), new(big.Int).Mul(denominator, inverse)),
		)
	}

	result := new(big.Int).Mul(prod0, inverse)
	return result, nil
}

func gt(x *big.Int, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return integer.One()
	}
	return integer.Zero()
}

func mulShiftRoundUp(x *big.Int, y *big.Int, offset uint8) (*big.Int, error) {
	result, err := mulShiftRoundDown(x, y, offset)
	if err != nil {
		return nil, err
	}
	v := new(big.Int).Mod(
		new(big.Int).Mul(x, y),
		new(big.Int).Lsh(integer.One(), uint(offset)),
	)
	if v.Cmp(integer.Zero()) != 0 {
		result = new(big.Int).Add(result, integer.One())
	}
	return result, nil
}

func mulShiftRoundDown(x *big.Int, y *big.Int, offset uint8) (*big.Int, error) {
	prod0, prod1 := getMulProds(x, y)
	result := big.NewInt(0)
	if prod0.Cmp(integer.Zero()) != 0 {
		result = new(big.Int).Rsh(prod0, uint(offset))
	}
	if prod1.Cmp(integer.Zero()) != 0 {
		if prod1.Cmp(new(big.Int).Lsh(integer.One(), uint(offset))) >= 0 {
			return nil, ErrMulShiftOverflow
		}
		result = new(big.Int).Add(
			result,
			new(big.Int).Lsh(prod1, uint(256-int(offset))),
		)
	}

	return result, nil
}

func getMulProds(x *big.Int, y *big.Int) (*big.Int, *big.Int) {
	not0 := maxUint256
	mm := new(big.Int).Mod(new(big.Int).Mul(x, y), not0)
	prod0 := new(big.Int).Mul(x, y)
	prod1 := new(big.Int).Sub(
		new(big.Int).Sub(mm, prod0),
		lt(mm, prod0),
	)
	return prod0, prod1
}

func lt(x *big.Int, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return integer.One()
	}
	return integer.Zero()
}

func bitwiseNotUint256(x *big.Int) *big.Int {
	return new(big.Int).Xor(x, maxUint256)
}

package math

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrXOutOfBounds       = errors.New("X_OUT_OF_BOUNDS")
	ErrYOutOfBounds       = errors.New("Y_OUT_OF_BOUNDS")
	ErrProductOutOfBounds = errors.New("PRODUCT_OUT_OF_BOUNDS")
	ErrInvalidExponent    = errors.New("INVALID_EXPONENT")
)

var LogExpMath *logExpMath

func init() {
	one_18 := integer.TenPow(18)
	one_20 := integer.TenPow(20)
	one_36 := integer.TenPow(36)

	LogExpMath = &logExpMath{
		ONE_18:               one_18,
		ONE_20:               one_20,
		ONE_36:               one_36,
		MILD_EXPONENT_BOUND:  new(uint256.Int).Div(new(uint256.Int).Exp(number.Number_2, uint256.NewInt(254)), number.TenPow(20)),
		LN_36_LOWER_BOUND:    new(big.Int).Sub(one_18, integer.TenPow(17)),
		LN_36_UPPER_BOUND:    new(big.Int).Add(one_18, integer.TenPow(17)),
		MIN_NATURAL_EXPONENT: new(big.Int).Mul(big.NewInt(-41), one_18),
		MAX_NATURAL_EXPONENT: new(big.Int).Mul(big.NewInt(130), one_18),
		x0:                   bignumber.NewBig10("128000000000000000000"),
		a0:                   bignumber.NewBig10("38877084059945950922200000000000000000000000000000000000"),
		x1:                   bignumber.NewBig10("64000000000000000000"),
		a1:                   bignumber.NewBig10("6235149080811616882910000000"),
		x2:                   bignumber.NewBig10("3200000000000000000000"),
		a2:                   bignumber.NewBig10("7896296018268069516100000000000000"),
		x3:                   bignumber.NewBig10("1600000000000000000000"),
		a3:                   bignumber.NewBig10("888611052050787263676000000"),
		x4:                   bignumber.NewBig10("800000000000000000000"),
		a4:                   bignumber.NewBig10("298095798704172827474000"),
		x5:                   bignumber.NewBig10("400000000000000000000"),
		a5:                   bignumber.NewBig10("5459815003314423907810"),
		x6:                   bignumber.NewBig10("200000000000000000000"),
		a6:                   bignumber.NewBig10("738905609893065022723"),
		x7:                   bignumber.NewBig10("100000000000000000000"),
		a7:                   bignumber.NewBig10("271828182845904523536"),
		x8:                   bignumber.NewBig10("50000000000000000000"),
		a8:                   bignumber.NewBig10("164872127070012814685"),
		x9:                   bignumber.NewBig10("25000000000000000000"),
		a9:                   bignumber.NewBig10("128402541668774148407"),
		x10:                  bignumber.NewBig10("12500000000000000000"),
		a10:                  bignumber.NewBig10("113314845306682631683"),
		x11:                  bignumber.NewBig10("6250000000000000000"),
		a11:                  bignumber.NewBig10("106449445891785942956"),
	}
}

type logExpMath struct {
	ONE_18               *big.Int
	ONE_20               *big.Int
	ONE_36               *big.Int
	MILD_EXPONENT_BOUND  *uint256.Int
	LN_36_LOWER_BOUND    *big.Int
	LN_36_UPPER_BOUND    *big.Int
	MIN_NATURAL_EXPONENT *big.Int
	MAX_NATURAL_EXPONENT *big.Int
	x0                   *big.Int
	a0                   *big.Int
	x1                   *big.Int
	a1                   *big.Int
	x2                   *big.Int
	a2                   *big.Int
	x3                   *big.Int
	a3                   *big.Int
	x4                   *big.Int
	a4                   *big.Int
	x5                   *big.Int
	a5                   *big.Int
	x6                   *big.Int
	a6                   *big.Int
	x7                   *big.Int
	a7                   *big.Int
	x8                   *big.Int
	a8                   *big.Int
	x9                   *big.Int
	a9                   *big.Int
	x10                  *big.Int
	a10                  *big.Int
	x11                  *big.Int
	a11                  *big.Int
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/LogExpMath.sol#L93
func (l *logExpMath) Pow(x *uint256.Int, y *uint256.Int) (*uint256.Int, error) {
	if y.Cmp(number.Zero) == 0 {
		return number.Number_1e18, nil
	}

	if x.Cmp(number.Zero) == 0 {
		return number.Zero, nil
	}

	if new(uint256.Int).Rsh(x, 255).Cmp(number.Zero) != 0 {
		return nil, ErrXOutOfBounds
	}
	x_int256 := x.ToBig()

	if y.Cmp(l.MILD_EXPONENT_BOUND) >= 0 {
		return nil, ErrYOutOfBounds
	}
	y_int256 := y.ToBig()

	var logXTimesY *big.Int

	if l.LN_36_LOWER_BOUND.Cmp(x_int256) < 0 && x_int256.Cmp(l.LN_36_UPPER_BOUND) < 0 {
		ln36X := l._ln_36(x_int256)
		logXTimesY = new(big.Int).Add(
			new(big.Int).Mul(new(big.Int).Quo(ln36X, l.ONE_18), y_int256),
			new(big.Int).Quo(new(big.Int).Mul(new(big.Int).Mod(ln36X, l.ONE_18), y_int256), l.ONE_18),
		)
	} else {
		logXTimesY = new(big.Int).Mul(l._ln(y_int256), y_int256)
	}

	logXTimesY = new(big.Int).Quo(logXTimesY, l.ONE_18)

	if l.MIN_NATURAL_EXPONENT.Cmp(logXTimesY) > 0 || logXTimesY.Cmp(l.MAX_NATURAL_EXPONENT) > 0 {
		return nil, ErrProductOutOfBounds
	}

	result, err := l.Exp(logXTimesY)
	if err != nil {
		return nil, err
	}

	return uint256.MustFromBig(result), nil
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/LogExpMath.sol#L146
func (l *logExpMath) Exp(x *big.Int) (*big.Int, error) {
	if x.Cmp(l.MIN_NATURAL_EXPONENT) < 0 || x.Cmp(l.MAX_NATURAL_EXPONENT) > 0 {
		return nil, ErrInvalidExponent
	}

	if x.Cmp(integer.Zero()) < 0 {
		negativeXExp, err := l.Exp(new(big.Int).Neg(x))
		if err != nil {
			return nil, err
		}

		return new(big.Int).Quo(new(big.Int).Mul(l.ONE_18, l.ONE_18), negativeXExp), nil
	}

	var firstAN *big.Int
	if x.Cmp(l.x0) >= 0 {
		x = new(big.Int).Sub(x, l.x0)
		firstAN = l.a0
	} else if x.Cmp(l.x1) >= 0 {
		x = new(big.Int).Sub(x, l.x1)
		firstAN = l.a1
	} else {
		firstAN = integer.One()
	}

	x = new(big.Int).Mul(x, big.NewInt(100))
	product := new(big.Int).Set(l.ONE_20)

	if x.Cmp(l.x2) >= 0 {
		x = new(big.Int).Sub(x, l.x2)
		product = new(big.Int).Quo(new(big.Int).Mul(product, l.a2), l.ONE_20)
	}

	if x.Cmp(l.x3) >= 0 {
		x = new(big.Int).Sub(x, l.x3)
		product = new(big.Int).Quo(new(big.Int).Mul(product, l.a3), l.ONE_20)
	}

	if x.Cmp(l.x4) >= 0 {
		x = new(big.Int).Sub(x, l.x4)
		product = new(big.Int).Quo(new(big.Int).Mul(product, l.a4), l.ONE_20)
	}

	if x.Cmp(l.x5) >= 0 {
		x = new(big.Int).Sub(x, l.x5)
		product = new(big.Int).Quo(new(big.Int).Mul(product, l.a5), l.ONE_20)
	}

	if x.Cmp(l.x6) >= 0 {
		x = new(big.Int).Sub(x, l.x6)
		product = new(big.Int).Quo(new(big.Int).Mul(product, l.a6), l.ONE_20)
	}

	if x.Cmp(l.x7) >= 0 {
		x = new(big.Int).Sub(x, l.x7)
		product = new(big.Int).Quo(new(big.Int).Mul(product, l.a7), l.ONE_20)
	}

	if x.Cmp(l.x8) >= 0 {
		x = new(big.Int).Sub(x, l.x8)
		product = new(big.Int).Quo(new(big.Int).Mul(product, l.a8), l.ONE_20)
	}

	if x.Cmp(l.x9) >= 0 {
		x = new(big.Int).Sub(x, l.x9)
		product = new(big.Int).Quo(new(big.Int).Mul(product, l.a9), l.ONE_20)
	}

	seriesSum := new(big.Int).Set(l.ONE_20)

	term := new(big.Int).Set(x)
	seriesSum = new(big.Int).Add(seriesSum, term)

	term = new(big.Int).Quo(new(big.Int).Quo(new(big.Int).Mul(term, x), l.ONE_20), big.NewInt(2))
	seriesSum = new(big.Int).Add(seriesSum, term)

	term = new(big.Int).Quo(new(big.Int).Quo(new(big.Int).Mul(term, x), l.ONE_20), big.NewInt(3))
	seriesSum = new(big.Int).Add(seriesSum, term)

	term = new(big.Int).Quo(new(big.Int).Quo(new(big.Int).Mul(term, x), l.ONE_20), big.NewInt(4))
	seriesSum = new(big.Int).Add(seriesSum, term)

	term = new(big.Int).Quo(new(big.Int).Quo(new(big.Int).Mul(term, x), l.ONE_20), big.NewInt(5))
	seriesSum = new(big.Int).Add(seriesSum, term)

	term = new(big.Int).Quo(new(big.Int).Quo(new(big.Int).Mul(term, x), l.ONE_20), big.NewInt(6))
	seriesSum = new(big.Int).Add(seriesSum, term)

	term = new(big.Int).Quo(new(big.Int).Quo(new(big.Int).Mul(term, x), l.ONE_20), big.NewInt(7))
	seriesSum = new(big.Int).Add(seriesSum, term)

	term = new(big.Int).Quo(new(big.Int).Quo(new(big.Int).Mul(term, x), l.ONE_20), big.NewInt(8))
	seriesSum = new(big.Int).Add(seriesSum, term)

	term = new(big.Int).Quo(new(big.Int).Quo(new(big.Int).Mul(term, x), l.ONE_20), big.NewInt(9))
	seriesSum = new(big.Int).Add(seriesSum, term)

	term = new(big.Int).Quo(new(big.Int).Quo(new(big.Int).Mul(term, x), l.ONE_20), big.NewInt(10))
	seriesSum = new(big.Int).Add(seriesSum, term)

	term = new(big.Int).Quo(new(big.Int).Quo(new(big.Int).Mul(term, x), l.ONE_20), big.NewInt(11))
	seriesSum = new(big.Int).Add(seriesSum, term)

	term = new(big.Int).Quo(new(big.Int).Quo(new(big.Int).Mul(term, x), l.ONE_20), big.NewInt(12))
	seriesSum = new(big.Int).Add(seriesSum, term)

	return new(big.Int).Quo(new(big.Int).Mul(new(big.Int).Quo(new(big.Int).Mul(product, seriesSum), l.ONE_20), firstAN), big.NewInt(100)), nil
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/LogExpMath.sol#L466
func (l *logExpMath) _ln_36(x *big.Int) *big.Int {
	x = new(big.Int).Mul(x, l.ONE_18)

	z := new(big.Int).Quo(
		new(big.Int).Mul(
			new(big.Int).Sub(x, l.ONE_36),
			l.ONE_36,
		),
		new(big.Int).Add(x, l.ONE_36),
	)
	zSquared := new(big.Int).Quo(new(big.Int).Mul(z, z), l.ONE_36)

	num := new(big.Int).Set(z)
	seriesSum := new(big.Int).Set(num)

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(3)))

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(5)))

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(7)))

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(9)))

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(11)))

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(13)))

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(15)))

	return new(big.Int).Mul(seriesSum, integer.Two())
}

// https://github.com/balancer/balancer-v2-monorepo/blob/c7d4abbea39834e7778f9ff7999aaceb4e8aa048/pkg/solidity-utils/contracts/math/LogExpMath.sol#L326
func (l *logExpMath) _ln(a *big.Int) *big.Int {
	if a.Cmp(l.ONE_18) < 0 {
		return new(big.Int).Neg(l._ln(new(big.Int).Quo(new(big.Int).Mul(l.ONE_18, l.ONE_18), a)))
	}

	sum := integer.Zero()
	if a.Cmp(new(big.Int).Mul(l.a0, l.ONE_18)) >= 0 {
		a = new(big.Int).Quo(a, l.a0)
		sum = new(big.Int).Add(sum, l.x0)
	}

	if a.Cmp(new(big.Int).Mul(l.a1, l.ONE_18)) >= 0 {
		a = new(big.Int).Quo(a, l.a1)
		sum = new(big.Int).Add(sum, l.x1)
	}

	sum = new(big.Int).Mul(sum, big.NewInt(100))
	a = new(big.Int).Mul(a, big.NewInt(100))

	if a.Cmp(l.a2) >= 0 {
		a = new(big.Int).Quo(new(big.Int).Mul(a, l.ONE_20), l.a2)
		sum = new(big.Int).Add(sum, l.x2)
	}

	if a.Cmp(l.a3) >= 0 {
		a = new(big.Int).Quo(new(big.Int).Mul(a, l.ONE_20), l.a3)
		sum = new(big.Int).Add(sum, l.x3)
	}

	if a.Cmp(l.a4) >= 0 {
		a = new(big.Int).Quo(new(big.Int).Mul(a, l.ONE_20), l.a4)
		sum = new(big.Int).Add(sum, l.x4)
	}

	if a.Cmp(l.a5) >= 0 {
		a = new(big.Int).Quo(new(big.Int).Mul(a, l.ONE_20), l.a5)
		sum = new(big.Int).Add(sum, l.x5)
	}

	if a.Cmp(l.a6) >= 0 {
		a = new(big.Int).Quo(new(big.Int).Mul(a, l.ONE_20), l.a6)
		sum = new(big.Int).Add(sum, l.x6)
	}

	if a.Cmp(l.a7) >= 0 {
		a = new(big.Int).Quo(new(big.Int).Mul(a, l.ONE_20), l.a7)
		sum = new(big.Int).Add(sum, l.x7)
	}

	if a.Cmp(l.a8) >= 0 {
		a = new(big.Int).Quo(new(big.Int).Mul(a, l.ONE_20), l.a8)
		sum = new(big.Int).Add(sum, l.x8)
	}

	if a.Cmp(l.a9) >= 0 {
		a = new(big.Int).Quo(new(big.Int).Mul(a, l.ONE_20), l.a9)
		sum = new(big.Int).Add(sum, l.x9)
	}

	if a.Cmp(l.a10) >= 0 {
		a = new(big.Int).Quo(new(big.Int).Mul(a, l.ONE_20), l.a10)
		sum = new(big.Int).Add(sum, l.x10)
	}

	if a.Cmp(l.a11) >= 0 {
		a = new(big.Int).Quo(new(big.Int).Mul(a, l.ONE_20), l.a11)
		sum = new(big.Int).Add(sum, l.x11)
	}

	z := new(big.Int).Quo(new(big.Int).Mul(new(big.Int).Sub(a, l.ONE_20), l.ONE_20), new(big.Int).Add(a, l.ONE_20))
	zSquared := new(big.Int).Quo(new(big.Int).Mul(z, z), l.ONE_20)

	num := z
	seriesSum := num

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_20)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(3)))

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_20)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(5)))

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_20)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(7)))

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_20)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(9)))

	num = new(big.Int).Quo(new(big.Int).Mul(num, zSquared), l.ONE_20)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Quo(num, big.NewInt(11)))

	seriesSum = new(big.Int).Mul(seriesSum, integer.Two())

	return new(big.Int).Quo(new(big.Int).Add(sum, seriesSum), big.NewInt(100))
}

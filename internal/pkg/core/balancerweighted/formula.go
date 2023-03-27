package balancerweighted

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
)

var ONE_18 = constant.TenPowInt(18)
var ONE_20 = constant.TenPowInt(20)
var ONE_36 = constant.TenPowInt(36)
var MAX_NATURAL_EXPONENT = new(big.Int).Mul(big.NewInt(130), ONE_18)
var MIN_NATURAL_EXPONENT = new(big.Int).Mul(big.NewInt(-41), ONE_18)
var LN_36_LOWER_BOUND = new(big.Int).Sub(ONE_18, constant.TenPowInt(17))
var LN_36_UPPER_BOUND = new(big.Int).Add(ONE_18, constant.TenPowInt(17))
var MILD_EXPONENT_BOUND = new(big.Int).Div(new(big.Int).Exp(constant.Two, big.NewInt(254), nil), ONE_20)
var x0 = utils.NewBig10("128000000000000000000")
var a0 = utils.NewBig10("38877084059945950922200000000000000000000000000000000000")
var x1 = utils.NewBig10("64000000000000000000")
var a1 = utils.NewBig10("6235149080811616882910000000")

var x2 = utils.NewBig10("3200000000000000000000")
var a2 = utils.NewBig10("7896296018268069516100000000000000")
var x3 = utils.NewBig10("1600000000000000000000")
var a3 = utils.NewBig10("888611052050787263676000000")
var x4 = utils.NewBig10("800000000000000000000")
var a4 = utils.NewBig10("298095798704172827474000")
var x5 = utils.NewBig10("400000000000000000000")
var a5 = utils.NewBig10("5459815003314423907810")
var x6 = utils.NewBig10("200000000000000000000")
var a6 = utils.NewBig10("738905609893065022723")
var x7 = utils.NewBig10("100000000000000000000")
var a7 = utils.NewBig10("271828182845904523536")
var x8 = utils.NewBig10("50000000000000000000")
var a8 = utils.NewBig10("164872127070012814685")
var x9 = utils.NewBig10("25000000000000000000")
var a9 = utils.NewBig10("128402541668774148407")
var x10 = utils.NewBig10("12500000000000000000")
var a10 = utils.NewBig10("113314845306682631683")
var x11 = utils.NewBig10("6250000000000000000")
var a11 = utils.NewBig10("106449445891785942956")

func init() {
}

func pow(x *big.Int, y *big.Int) *big.Int {
	if y.Cmp(constant.Zero) == 0 {
		return ONE_18
	}
	if x.Cmp(constant.Zero) == 0 {
		return constant.Zero
	}
	var logx_times_y *big.Int
	if LN_36_LOWER_BOUND.Cmp(x) < 0 && x.Cmp(LN_36_UPPER_BOUND) < 0 {
		var ln_36_x = _ln_36(x)
		logx_times_y = new(big.Int).Add(
			new(big.Int).Mul(new(big.Int).Div(ln_36_x, ONE_18), y),
			new(big.Int).Div(new(big.Int).Mul(new(big.Int).Mod(ln_36_x, ONE_18), y), ONE_18),
		)
	} else {
		logx_times_y = new(big.Int).Mul(_ln(x), y)
	}
	logx_times_y = new(big.Int).Div(logx_times_y, ONE_18)
	return exp(logx_times_y)
}

func exp(x *big.Int) *big.Int {
	if x.Cmp(constant.Zero) < 0 {
		var temp = exp(new(big.Int).Neg(x))
		return new(big.Int).Div(new(big.Int).Mul(ONE_18, ONE_18), temp)
	}
	var firstAN *big.Int
	if x.Cmp(x0) >= 0 {
		x = new(big.Int).Sub(x, x0)
		firstAN = a0
	} else if x.Cmp(x1) >= 0 {
		x = new(big.Int).Sub(x, x1)
		firstAN = a1
	} else {
		firstAN = constant.One
	}
	x = new(big.Int).Mul(x, constant.TenPowInt(2))
	var product = ONE_20
	if x.Cmp(x2) >= 0 {
		x = new(big.Int).Sub(x, x2)
		product = new(big.Int).Div(new(big.Int).Mul(product, a2), ONE_20)
	}
	if x.Cmp(x3) >= 0 {
		x = new(big.Int).Sub(x, x3)
		product = new(big.Int).Div(new(big.Int).Mul(product, a3), ONE_20)
	}
	if x.Cmp(x4) >= 0 {
		x = new(big.Int).Sub(x, x4)
		product = new(big.Int).Div(new(big.Int).Mul(product, a4), ONE_20)
	}
	if x.Cmp(x5) >= 0 {
		x = new(big.Int).Sub(x, x5)
		product = new(big.Int).Div(new(big.Int).Mul(product, a5), ONE_20)
	}
	if x.Cmp(x6) >= 0 {
		x = new(big.Int).Sub(x, x6)
		product = new(big.Int).Div(new(big.Int).Mul(product, a6), ONE_20)
	}
	if x.Cmp(x7) >= 0 {
		x = new(big.Int).Sub(x, x7)
		product = new(big.Int).Div(new(big.Int).Mul(product, a7), ONE_20)
	}
	if x.Cmp(x8) >= 0 {
		x = new(big.Int).Sub(x, x8)
		product = new(big.Int).Div(new(big.Int).Mul(product, a8), ONE_20)
	}
	if x.Cmp(x9) >= 0 {
		x = new(big.Int).Sub(x, x9)
		product = new(big.Int).Div(new(big.Int).Mul(product, a9), ONE_20)
	}
	var seriesSum = ONE_20
	var term = x
	seriesSum = new(big.Int).Add(seriesSum, term)
	term = new(big.Int).Div(new(big.Int).Div(new(big.Int).Mul(term, x), ONE_20), constant.Two)
	seriesSum = new(big.Int).Add(seriesSum, term)
	term = new(big.Int).Div(new(big.Int).Div(new(big.Int).Mul(term, x), ONE_20), big.NewInt(3))
	seriesSum = new(big.Int).Add(seriesSum, term)
	term = new(big.Int).Div(new(big.Int).Div(new(big.Int).Mul(term, x), ONE_20), big.NewInt(4))
	seriesSum = new(big.Int).Add(seriesSum, term)
	term = new(big.Int).Div(new(big.Int).Div(new(big.Int).Mul(term, x), ONE_20), big.NewInt(5))
	seriesSum = new(big.Int).Add(seriesSum, term)
	term = new(big.Int).Div(new(big.Int).Div(new(big.Int).Mul(term, x), ONE_20), big.NewInt(6))
	seriesSum = new(big.Int).Add(seriesSum, term)
	term = new(big.Int).Div(new(big.Int).Div(new(big.Int).Mul(term, x), ONE_20), big.NewInt(7))
	seriesSum = new(big.Int).Add(seriesSum, term)
	term = new(big.Int).Div(new(big.Int).Div(new(big.Int).Mul(term, x), ONE_20), big.NewInt(8))
	seriesSum = new(big.Int).Add(seriesSum, term)
	term = new(big.Int).Div(new(big.Int).Div(new(big.Int).Mul(term, x), ONE_20), big.NewInt(9))
	seriesSum = new(big.Int).Add(seriesSum, term)
	term = new(big.Int).Div(new(big.Int).Div(new(big.Int).Mul(term, x), ONE_20), big.NewInt(10))
	seriesSum = new(big.Int).Add(seriesSum, term)
	term = new(big.Int).Div(new(big.Int).Div(new(big.Int).Mul(term, x), ONE_20), big.NewInt(11))
	seriesSum = new(big.Int).Add(seriesSum, term)
	term = new(big.Int).Div(new(big.Int).Div(new(big.Int).Mul(term, x), ONE_20), big.NewInt(12))
	seriesSum = new(big.Int).Add(seriesSum, term)
	return new(big.Int).Div(new(big.Int).Mul(new(big.Int).Div(new(big.Int).Mul(product, seriesSum), ONE_20), firstAN), constant.TenPowInt(2))
}

func _ln(a *big.Int) *big.Int {
	if a.Cmp(ONE_18) < 0 {
		var temp = _ln(new(big.Int).Div(new(big.Int).Mul(ONE_18, ONE_18), a))
		return new(big.Int).Neg(temp)
	}
	var sum = constant.Zero
	if a.Cmp(new(big.Int).Mul(a0, ONE_18)) >= 0 {
		a = new(big.Int).Div(a, a0)
		sum = new(big.Int).Add(sum, x0)
	}
	if a.Cmp(new(big.Int).Mul(a1, ONE_18)) >= 0 {
		a = new(big.Int).Div(a, a1)
		sum = new(big.Int).Add(sum, x1)
	}
	sum = new(big.Int).Mul(sum, constant.TenPowInt(2))
	a = new(big.Int).Mul(a, constant.TenPowInt(2))

	if a.Cmp(a2) >= 0 {
		a = new(big.Int).Div(new(big.Int).Mul(a, ONE_20), a2)
		sum = new(big.Int).Add(sum, x2)
	}
	if a.Cmp(a3) >= 0 {
		a = new(big.Int).Div(new(big.Int).Mul(a, ONE_20), a3)
		sum = new(big.Int).Add(sum, x3)
	}
	if a.Cmp(a4) >= 0 {
		a = new(big.Int).Div(new(big.Int).Mul(a, ONE_20), a4)
		sum = new(big.Int).Add(sum, x4)
	}
	if a.Cmp(a5) >= 0 {
		a = new(big.Int).Div(new(big.Int).Mul(a, ONE_20), a5)
		sum = new(big.Int).Add(sum, x5)
	}
	if a.Cmp(a6) >= 0 {
		a = new(big.Int).Div(new(big.Int).Mul(a, ONE_20), a6)
		sum = new(big.Int).Add(sum, x6)
	}
	if a.Cmp(a7) >= 0 {
		a = new(big.Int).Div(new(big.Int).Mul(a, ONE_20), a7)
		sum = new(big.Int).Add(sum, x7)
	}
	if a.Cmp(a8) >= 0 {
		a = new(big.Int).Div(new(big.Int).Mul(a, ONE_20), a8)
		sum = new(big.Int).Add(sum, x8)
	}
	if a.Cmp(a9) >= 0 {
		a = new(big.Int).Div(new(big.Int).Mul(a, ONE_20), a9)
		sum = new(big.Int).Add(sum, x9)
	}
	if a.Cmp(a10) >= 0 {
		a = new(big.Int).Div(new(big.Int).Mul(a, ONE_20), a10)
		sum = new(big.Int).Add(sum, x10)
	}
	if a.Cmp(a11) >= 0 {
		a = new(big.Int).Div(new(big.Int).Mul(a, ONE_20), a11)
		sum = new(big.Int).Add(sum, x11)
	}
	var z = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(a, ONE_20), ONE_20), new(big.Int).Add(a, ONE_20))
	var z_squared = new(big.Int).Div(new(big.Int).Mul(z, z), ONE_20)
	var num = z
	var seriesSum = num
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_20)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(3)))
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_20)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(5)))
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_20)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(7)))
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_20)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(9)))
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_20)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(11)))
	seriesSum = new(big.Int).Mul(seriesSum, constant.Two)
	return new(big.Int).Div(new(big.Int).Add(sum, seriesSum), constant.TenPowInt(2))
}

func _ln_36(x *big.Int) *big.Int {
	x = new(big.Int).Mul(x, ONE_18)
	var z = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(x, ONE_36), ONE_36), new(big.Int).Add(x, ONE_36))
	var z_squared = new(big.Int).Div(new(big.Int).Mul(z, z), ONE_36)
	var num = z
	var seriesSum = num
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(3)))
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(5)))
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(7)))
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(9)))
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(11)))
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(13)))
	num = new(big.Int).Div(new(big.Int).Mul(num, z_squared), ONE_36)
	seriesSum = new(big.Int).Add(seriesSum, new(big.Int).Div(num, big.NewInt(15)))
	return new(big.Int).Mul(seriesSum, constant.Two)
}

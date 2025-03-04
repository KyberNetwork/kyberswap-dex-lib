package pandafun

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var ROUNDING_UP = true
var ROUNDING_DOWN = false

var PRICE_SCALE = bignumber.TenPowInt(36)
var FEE_SCALE = bignumber.TenPowInt(4)

func mulDiv(x, y, denominator *big.Int, isRoundingUp bool) *big.Int {
	var tmp big.Int

	mul := tmp.Mul(x, y)
	res := new(big.Int).Div(mul, denominator)

	if isRoundingUp && tmp.Mod(mul, denominator).Sign() > 0 {
		res.Add(res, bignumber.One)
	}

	return res
}

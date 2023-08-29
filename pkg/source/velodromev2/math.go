package velodromev2

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// reference from smart-contract:
//
//	function getAmountOut(uint amountIn, address tokenIn) external view returns (uint)
//	https://optimistic.etherscan.io/address/0x25cbddb98b35ab1ff77413456b31ec81a6b6b746#code
func getAmountOut(
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	decimalIn *big.Int,
	decimalOut *big.Int,
	swapFee *big.Int,
	stable bool,
) *big.Int {
	var amountAfterFee = calAmountAfterFee(amountIn, swapFee)
	if amountAfterFee.Cmp(bignumber.ZeroBI) <= 0 {
		return bignumber.ZeroBI
	}

	return getExactQuote(amountAfterFee, reserveIn, reserveOut, decimalIn, decimalOut, stable)
}

func getExactQuote(
	amountIn *big.Int,
	reserveIn *big.Int,
	reserveOut *big.Int,
	decimalIn *big.Int,
	decimalOut *big.Int,
	stable bool,
) *big.Int {
	amountOut := big.NewInt(0)

	if amountIn.Cmp(bignumber.ZeroBI) <= 0 {
		return amountOut
	}

	if stable {
		xy := _k(reserveIn, reserveOut, decimalIn, decimalOut, stable)
		_reserveIn := new(big.Int).Div(new(big.Int).Mul(reserveIn, bignumber.BONE), decimalIn)
		_reserveOut := new(big.Int).Div(new(big.Int).Mul(reserveOut, bignumber.BONE), decimalOut)
		_amountIn := new(big.Int).Div(new(big.Int).Mul(amountIn, bignumber.BONE), decimalIn)

		y := new(big.Int).Sub(_reserveOut, _get_y(new(big.Int).Add(_amountIn, _reserveIn), xy, _reserveOut))

		amountOut = new(big.Int).Div(new(big.Int).Mul(y, decimalOut), bignumber.BONE)
	} else {
		// (x*In)/(y+In)
		numerator := new(big.Int).Mul(amountIn, reserveOut)
		denominator := new(big.Int).Add(reserveIn, amountIn)

		if denominator.Cmp(bignumber.ZeroBI) > 0 {
			amountOut = new(big.Int).Div(numerator, denominator)
		}
	}

	if !validateAmountOut(amountIn, amountOut, reserveIn, reserveOut, decimalIn, decimalOut, stable) {
		return bignumber.ZeroBI
	}

	return amountOut
}

func calAmountAfterFee(amountIn, swapFee *big.Int) *big.Int {
	// In - (fee*In)/Bone
	return new(big.Int).Sub(amountIn, new(big.Int).Div(new(big.Int).Mul(swapFee, amountIn), bignumber.BONE))
}

func _k(x, y, decimals0, decimals1 *big.Int, stable bool) *big.Int {
	if stable {
		_x := new(big.Int).Div(new(big.Int).Mul(x, bignumber.BONE), decimals0)
		_y := new(big.Int).Div(new(big.Int).Mul(y, bignumber.BONE), decimals1)

		_a := new(big.Int).Div(new(big.Int).Mul(_x, _y), bignumber.BONE)
		_b := new(big.Int).Div(new(big.Int).Add(new(big.Int).Mul(_x, _x), new(big.Int).Mul(_y, _y)), bignumber.BONE)

		// x3y+y3x >= k
		return new(big.Int).Div(new(big.Int).Mul(_a, _b), bignumber.BONE)
	} else {
		// xy >= k
		return new(big.Int).Mul(x, y)
	}
}

func _get_y(x0, xy, y *big.Int) *big.Int {
	_y := new(big.Int).Set(y)

	for i := 0; i < 255; i++ {
		y_prev := new(big.Int).Set(_y)

		k := _f(x0, _y)
		d := _d(x0, _y)
		if d.Cmp(bignumber.ZeroBI) <= 0 {
			return bignumber.ZeroBI
		}

		if k.Cmp(xy) < 0 {
			dy := new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(xy, k), bignumber.BONE), d)
			_y.Add(_y, dy)
		} else {
			dy := new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(k, xy), bignumber.BONE), d)
			_y.Sub(_y, dy)
		}

		diff := new(big.Int).Sub(_y, y_prev)
		if diff.CmpAbs(big.NewInt(1)) <= 0 {
			return _y
		}
	}

	return _y
}

func _f(x0, y *big.Int) *big.Int {
	// y^3*x
	y3x := new(big.Int).Mul(new(big.Int).Mul(y, y), new(big.Int).Mul(y, x0))
	// x^3*y
	x3y := new(big.Int).Mul(new(big.Int).Mul(x0, x0), new(big.Int).Mul(x0, y))

	numerator := new(big.Int).Add(y3x, x3y)
	denominator := new(big.Int).Mul(new(big.Int).Mul(bignumber.BONE, bignumber.BONE), bignumber.BONE)

	return new(big.Int).Div(numerator, denominator)
}

func _d(x0, y *big.Int) *big.Int {
	// 3*x*y^2 + x^3
	numerator := new(big.Int).Add(new(big.Int).Mul(new(big.Int).Mul(bignumber.Three, x0), new(big.Int).Mul(y, y)), new(big.Int).Mul(new(big.Int).Mul(x0, x0), x0))
	denominator := new(big.Int).Mul(bignumber.BONE, bignumber.BONE)

	return new(big.Int).Div(numerator, denominator)
}

// The SC required `K` after swap with condition:
//
//	require(_k(_balance0, _balance1) >= _k(_reserve0, _reserve1), 'K');
//
// validateAmountOut to check if after swap, the condition still valid.
func validateAmountOut(amountIn, amountOut, reserveIn, reserveOut, decimalIn, decimalOut *big.Int, stable bool) bool {
	balanceIn := new(big.Int).Add(reserveIn, amountIn)
	balanceOut := new(big.Int).Sub(reserveOut, amountOut)

	return _k(balanceIn, balanceOut, decimalIn, decimalOut, stable).
		Cmp(_k(reserveIn, reserveOut, decimalIn, decimalOut, stable)) >= 0
}

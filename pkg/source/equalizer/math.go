package equalizer

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var errZeroDerivative = errors.New("zero derivative in _get_y")

func getAmountOut(
	amountIn *uint256.Int,
	reserveIn *uint256.Int,
	reserveOut *uint256.Int,
	decimalIn *uint256.Int,
	decimalOut *uint256.Int,
	swapFee *uint256.Int,
	stable bool,
) (*uint256.Int, error) {
	amountAfterFee := calAmountAfterFee(amountIn, swapFee)
	if amountAfterFee.IsZero() {
		return uint256.NewInt(0), nil
	}

	return getExactQuote(amountAfterFee, reserveIn, reserveOut, decimalIn, decimalOut, stable)
}

func getExactQuote(
	amountIn *uint256.Int,
	reserveIn *uint256.Int,
	reserveOut *uint256.Int,
	decimalIn *uint256.Int,
	decimalOut *uint256.Int,
	stable bool,
) (*uint256.Int, error) {
	if amountIn.IsZero() {
		return uint256.NewInt(0), nil
	}

	if stable {
		xy := _k(reserveIn, reserveOut, decimalIn, decimalOut)
		var _reserveIn, _reserveOut uint256.Int
		_reserveIn.Div(_reserveIn.Mul(reserveIn, big256.BONE), decimalIn)
		_reserveOut.Div(_reserveOut.Mul(reserveOut, big256.BONE), decimalOut)

		var _amountIn uint256.Int
		_amountIn.Div(_amountIn.Mul(amountIn, big256.BONE), decimalIn)

		_reserveIn.Add(&_amountIn, &_reserveIn)
		y, err := _get_y(&_reserveIn, xy, &_reserveOut)
		if err != nil {
			return nil, err
		}

		var amountOut uint256.Int
		amountOut.Sub(&_reserveOut, y)
		amountOut.Div(amountOut.Mul(&amountOut, decimalOut), big256.BONE)

		if !validateAmountOut(amountIn, &amountOut, reserveIn, reserveOut, decimalIn, decimalOut) {
			return uint256.NewInt(0), nil
		}
		return &amountOut, nil
	}

	// (amountIn * reserveOut) / (reserveIn + amountIn)
	var amountOut, denom uint256.Int
	denom.Add(reserveIn, amountIn)
	if denom.IsZero() {
		return uint256.NewInt(0), nil
	}
	amountOut.Div(amountOut.Mul(amountIn, reserveOut), &denom)

	if !validateAmountOutVolatile(amountIn, &amountOut, reserveIn, reserveOut) {
		return uint256.NewInt(0), nil
	}
	return &amountOut, nil
}

func calAmountAfterFee(amountIn, swapFee *uint256.Int) *uint256.Int {
	var fee uint256.Int
	fee.Div(fee.Mul(swapFee, amountIn), big256.BONE)
	result := new(uint256.Int).Sub(amountIn, &fee)
	return result
}

// _k computes the stable invariant: x³y + xy³ (normalized by decimals and BONE)
// SC: _a = (_x * _y) / 1e18; _b = _x*_x/1e18 + _y*_y/1e18; return _a*_b/1e18
func _k(x, y, decimals0, decimals1 *uint256.Int) *uint256.Int {
	var _x, _y, _a uint256.Int
	_x.Div(_x.Mul(x, big256.BONE), decimals0)
	_y.Div(_y.Mul(y, big256.BONE), decimals1)
	_a.Div(_a.Mul(&_x, &_y), big256.BONE)
	_b := _x.Add(
		_x.Div(_x.Mul(&_x, &_x), big256.BONE),
		_y.Div(_y.Mul(&_y, &_y), big256.BONE),
	)
	return _a.Div(_a.Mul(&_a, _b), big256.BONE)
}

func _get_y(x0, xy, y *uint256.Int) (*uint256.Int, error) {
	y = y.Clone()
	var dy uint256.Int
	for range 255 {
		k := _f(x0, y)
		d := _d(x0, y)
		if d.IsZero() {
			return nil, errZeroDerivative
		}

		if k.Cmp(xy) < 0 {
			dy.Sub(xy, k)
			dy.Div(dy.Mul(&dy, big256.BONE), d)
			y.Add(y, &dy)
		} else {
			dy.Sub(k, xy)
			dy.Div(dy.Mul(&dy, big256.BONE), d)
			y.Sub(y, &dy)
		}

		if dy.CmpUint64(1) <= 0 {
			return y, nil
		}
	}
	return y, nil
}

// _f computes x0*y³/BONE³ + x0³*y/BONE³
// SC: x0*(y*y/1e18*y/1e18)/1e18 + (x0*x0/1e18*x0/1e18)*y/1e18
func _f(x0, y *uint256.Int) *uint256.Int {
	var a, b uint256.Int
	// part1 = x0 * (y²/BONE) * y / BONE / BONE = x0*y³/BONE³
	a.Div(a.Mul(y, y), big256.BONE)
	a.Div(a.Mul(&a, y), big256.BONE)
	a.Div(a.Mul(x0, &a), big256.BONE)

	// part2 = (x0²/BONE * x0/BONE) * y / BONE = x0³*y/BONE³
	b.Div(b.Mul(x0, x0), big256.BONE)
	b.Div(b.Mul(&b, x0), big256.BONE)
	b.Div(b.Mul(&b, y), big256.BONE)

	return a.Add(&a, &b)
}

// _d computes 3*x0*y²/BONE² + x0³/BONE²
// SC: 3*x0*(y*y/1e18)/1e18 + (x0*x0/1e18*x0/1e18)
func _d(x0, y *uint256.Int) *uint256.Int {
	var a, b uint256.Int
	// b = y²/BONE (temp)
	b.Div(b.Mul(y, y), big256.BONE)
	// part1 = 3*x0*y²/BONE
	a.Mul(big256.U3, x0)
	a.Div(a.Mul(&a, &b), big256.BONE)

	// part2 = x0²/BONE * x0/BONE = x0³/BONE²
	b.Div(b.Mul(x0, x0), big256.BONE)
	b.Div(b.Mul(&b, x0), big256.BONE)

	return a.Add(&a, &b)
}

// validateAmountOut checks that k after swap >= k before swap (stable pools only)
func validateAmountOut(amountIn, amountOut, reserveIn, reserveOut, decimalIn, decimalOut *uint256.Int) bool {
	var balanceIn, balanceOut uint256.Int
	balanceIn.Add(reserveIn, amountIn)
	if balanceOut.Sub(reserveOut, amountOut); amountOut.Cmp(reserveOut) > 0 {
		return false
	}
	kAfter := _k(&balanceIn, &balanceOut, decimalIn, decimalOut)
	kBefore := _k(reserveIn, reserveOut, decimalIn, decimalOut)
	return kAfter.Cmp(kBefore) >= 0
}

// validateAmountOutVolatile checks that k after swap >= k before swap (volatile pools)
func validateAmountOutVolatile(amountIn, amountOut, reserveIn, reserveOut *uint256.Int) bool {
	if amountOut.Cmp(reserveOut) >= 0 {
		return false
	}
	var balanceIn, balanceOut uint256.Int
	balanceIn.Add(reserveIn, amountIn)
	balanceOut.Sub(reserveOut, amountOut)
	kAfter := new(uint256.Int).Mul(&balanceIn, &balanceOut)
	kBefore := new(uint256.Int).Mul(reserveIn, reserveOut)
	return kAfter.Cmp(kBefore) >= 0
}

package v2

import (
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func _Sqrt(a *uint256.Int, roundingUp bool) *uint256.Int {
	if a.IsZero() {
		return uint256.NewInt(0)
	}

	var result uint256.Int
	result.Sqrt(a)

	if roundingUp {
		var square uint256.Int
		square.Mul(&result, &result)
		if square.Lt(a) {
			result.AddUint64(&result, 1)
		}
	}

	return &result
}

func _MulShiftUp(a, b *uint256.Int, shift uint) *uint256.Int {
	res := new(uint256.Int).Mul(a, b)
	if shift == 0 {
		return res
	}
	divisor := new(uint256.Int).Lsh(big256.U1, shift)
	val, _ := _DivUp(res, divisor)
	return val
}

func _MulShift(a, b *uint256.Int, shift uint) *uint256.Int {
	res := new(uint256.Int).Mul(a, b)
	if shift == 0 {
		return res
	}
	res.Rsh(res, shift)
	return res
}

func _MulDivUpAlt(a, b, c *uint256.Int) *uint256.Int {
	if c.IsZero() {
		return uint256.NewInt(0)
	}
	res := new(uint256.Int).Mul(a, b)
	val, _ := _DivUp(res, c)
	return val
}

func _DivUp(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, shared.ErrDivisionByZero
	}
	var result uint256.Int
	result.Div(a, b)
	if new(uint256.Int).Mod(a, b).IsZero() {
		return &result, nil
	}
	return result.AddUint64(&result, 1), nil
}

func _F(x, px, py, x0, y0, c *uint256.Int) (*uint256.Int, error) {
	if c.Eq(big256.BONE) {
		if x.Eq(x0) {
			return y0.Clone(), nil
		}
		var v uint256.Int
		if x.Lt(x0) {
			v.Sub(x0, x).Mul(&v, px)
			val, _ := _DivUp(&v, py)
			output := new(uint256.Int).Add(y0, val)
			if output.Gt(shared.MaxUint112) {
				return big256.UMax.Clone(), nil
			}
			return output, nil
		} else {
			v.Sub(x, x0).Mul(&v, px)
			val := new(uint256.Int).Div(&v, py)
			return shared.SubTill0(y0, val), nil
		}
	}

	var a, b, d uint256.Int
	a.Sub(x0, x).Mul(&a, px)
	b.Mul(c, x).Add(&b, new(uint256.Int).Mul(new(uint256.Int).Sub(big256.BONE, c), x0))
	d.Mul(big256.BONE, x).Mul(&d, py)

	v, err := v3Utils.MulDivRoundingUp(&a, &b, &d)
	if err != nil {
		return nil, err
	}

	output := new(uint256.Int).Add(y0, v)
	if output.Gt(shared.MaxUint112) {
		return big256.UMax.Clone(), nil
	}
	return output, nil
}

func _FInverse(y, px, py, x0, y0, cx *uint256.Int) (*uint256.Int, error) {
	if cx.Eq(big256.BONE) {
		if y.Eq(y0) {
			return x0.Clone(), nil
		}
		var v uint256.Int
		if y.Lt(y0) {
			v.Sub(y0, y).Mul(&v, py)
			val, _ := _DivUp(&v, px)
			return new(uint256.Int).Add(x0, val), nil
		} else {
			v.Sub(y, y0).Mul(&v, py)
			val := new(uint256.Int).Div(&v, px)
			return shared.SubTill0(x0, val), nil
		}
	}

	var absB uint256.Int
	var sign bool
	{
		var term1, term2 uint256.Int
		// term1 = 1e18 * ((y - y0) * py + x0 * px)
		term1.Sub(y, y0).Mul(&term1, py).Add(&term1, new(uint256.Int).Mul(x0, px)).Mul(&term1, big256.BONE)

		// term2 = (cx << 1) * x0 * px
		term2.Lsh(cx, 1).Mul(&term2, x0).Mul(&term2, px)

		if term1.Lt(&term2) {
			absB.Sub(&term2, &term1)
			sign = true
		} else {
			absB.Sub(&term1, &term2)
			sign = false
		}

		if sign {
			diffMod := new(uint256.Int).Mod(&absB, px)
			absB.Div(&absB, px)
			if !diffMod.IsZero() {
				absB.AddUint64(&absB, 1)
			}
		} else {
			absB.Div(&absB, px)
		}
	}

	shift := uint(0)
	{
		shiftSquaredB := uint(0)
		if absB.BitLen() > 127 {
			shiftSquaredB = uint(absB.BitLen() - 127)
		}
		shiftFourAc := uint(0)
		val := new(uint256.Int).Mul(x0, shared.RA)
		if val.BitLen() > 109 {
			shiftFourAc = uint(val.BitLen() - 109)
		}
		if shiftSquaredB > shiftFourAc {
			shift = shiftSquaredB
		} else {
			shift = shiftFourAc
		}
	}
	twoShift := shift << 1

	var x *uint256.Int
	if sign {
		var fourAC uint256.Int
		// (cx * (1e18 - cx) << 2).unsafeMulShiftUp(x0 * x0, twoShift)
		fourAC.Sub(big256.BONE, cx).Mul(&fourAC, cx).Lsh(&fourAC, 2)
		fourAC.Mul(&fourAC, new(uint256.Int).Mul(x0, x0))
		divisor := new(uint256.Int).Lsh(big256.U1, twoShift)
		val, _ := _DivUp(&fourAC, divisor)
		fourAC.Set(val)

		squaredB := _MulShiftUp(&absB, &absB, twoShift)
		var discriminant uint256.Int
		discriminant.Add(squaredB, &fourAC)

		sqrt := _Sqrt(&discriminant, true)
		sqrt.Lsh(sqrt, shift)

		x = new(uint256.Int).Add(&absB, sqrt)
		x.Div(x, new(uint256.Int).Lsh(cx, 1))
	} else {
		var fourAC uint256.Int
		fourAC.Sub(big256.BONE, cx).Mul(&fourAC, cx).Lsh(&fourAC, 2)
		fourAC.Set(_MulShift(&fourAC, new(uint256.Int).Mul(x0, x0), twoShift))

		squaredB := _MulShift(&absB, &absB, twoShift)
		var discriminant uint256.Int
		discriminant.Add(squaredB, &fourAC)

		sqrt := _Sqrt(&discriminant, false)
		sqrt.Lsh(sqrt, shift)

		// x = ((1e18 - cx) << 1).unsafeMulDivUpAlt(x0 * x0, absB + sqrt)
		numerator := new(uint256.Int).Sub(big256.BONE, cx)
		numerator.Lsh(numerator, 1).Mul(numerator, new(uint256.Int).Mul(x0, x0))
		denominator := new(uint256.Int).Add(&absB, sqrt)

		x = _MulDivUpAlt(numerator, big256.U1, denominator)
	}

	if x.Gt(x0) {
		x.SubUint64(x, 1)
	}

	return x, nil
}

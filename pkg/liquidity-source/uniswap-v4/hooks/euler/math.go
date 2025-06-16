package euler

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/blockchain-toolkit/i256"
	"github.com/KyberNetwork/int256"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
)

var (
	e36        = big256.TenPow(36)
	maxUint112 = new(uint256.Int).SubUint64(new(uint256.Int).Lsh(big256.U1, 112), 1) // 2^112 - 1
	e18Int     = int256.NewInt(1e18)                                                 // 1e18
)

func _Sqrt(a *uint256.Int, roundingUp bool) (*uint256.Int, error) {
	if a.IsZero() {
		return uint256.NewInt(0), nil
	}

	var result uint256.Int
	result.Sqrt(a)

	if roundingUp {
		var square uint256.Int
		square.Mul(&result, &result)
		if square.Cmp(a) < 0 {
			result.Add(&result, big256.U1)
		}
	}

	return &result, nil
}

func _CeilDiv(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrDivisionByZero
	}

	if a.IsZero() {
		return uint256.NewInt(0), nil
	}

	var result uint256.Int
	result.SubUint64(a, 1)
	result.Div(&result, b)
	result.AddUint64(&result, 1)

	return &result, nil
}

// / @dev EulerSwap curve
// / @notice Computes the output `y` for a given input `x`.
// / @param x The input reserve value, constrained to 1 <= x <= x0.
// / @param px (1 <= px <= 1e25).
// / @param py (1 <= py <= 1e25).
// / @param x0 (1 <= x0 <= 2^112 - 1).
// / @param y0 (0 <= y0 <= 2^112 - 1).
// / @param c (0 <= c <= 1e18).
// / @return y The output reserve value corresponding to input `x`, guaranteed to satisfy `y0 <= y <= 2^112 - 1`.
func f(x, px, py, x0, y0, c *uint256.Int) (*uint256.Int, error) {
	var tmp1, tmp2, tmp3 uint256.Int

	tmp1.Sub(x0, x)
	tmp1.Mul(&tmp1, px)

	tmp2.Mul(c, x)

	tmp3.Sub(big256.BONE, c)
	tmp3.Mul(&tmp3, x0)
	tmp2.Add(&tmp2, &tmp3)

	tmp3.Mul(x, big256.BONE)

	v, err := v3Utils.MulDivRoundingUp(&tmp1, &tmp2, &tmp3)
	if err != nil {
		return nil, err
	}

	tmp1.SubUint64(py, 1)
	v.Add(v, &tmp1)
	v.Div(v, py)
	v.Add(v, y0)

	return v, nil
}

// / @dev EulerSwap inverse curve
// / @notice Computes the output `x` for a given input `y`.
// / @param y The input reserve value, constrained to y0 <= y <= 2^112 - 1.
// / @param px (1 <= px <= 1e25).
// / @param py (1 <= py <= 1e25).
// / @param x0 (1 <= x0 <= 2^112 - 1).
// / @param y0 (0 <= y0 <= 2^112 - 1).
// / @param c (0 <= c <= 1e18).
// / @return x The output reserve value corresponding to input `y`, guaranteed to satisfy `1 <= x <= x0`.
func fInverse(y, px, py, x0, y0, c *uint256.Int) (*uint256.Int, error) {
	// term1 = int256(Math.mulDiv(py * 1e18, y - y0, px, Math.Rounding.Ceil))
	term1 := new(uint256.Int).Mul(py, big256.BONE)

	var tmp uint256.Int
	tmp.Sub(y, y0)

	term1, overflow := term1.MulDivOverflow(term1, &tmp, px)
	if overflow {
		return nil, ErrOverflow
	}

	// term2 = (2 * int256(c) - int256(1e18)) * int256(x0)
	var term2 uint256.Int
	term2.Mul(c, big256.U2)
	term2.Sub(&term2, big256.BONE)
	term2.Mul(&term2, x0)

	// B = (term1 - term2) / int256(1e18)
	B := i256.SafeToInt256(term1)
	B.Sub(B, i256.SafeToInt256(&term2))
	B.Quo(B, e18Int)

	// C = Math.mulDiv(1e18 - c, x0 * x0, 1e18, Math.Rounding.Ceil)
	tmp.Sub(big256.BONE, c)
	term2.Mul(x0, x0)
	C, err := v3Utils.MulDivRoundingUp(&tmp, &term2, big256.BONE)
	if err != nil {
		return nil, err
	}

	// fourAC = Math.mulDiv(4 * c, C, 1e18, Math.Rounding.Ceil)
	tmp.Mul(c, big256.U4)
	fourAC, err := v3Utils.MulDivRoundingUp(&tmp, C, big256.BONE)
	if err != nil {
		return nil, err
	}

	var absB uint256.Int
	bigB := B.ToBig()
	bigB.Abs(bigB)
	absB.SetFromBig(bigB)

	var sqrt *uint256.Int
	if absB.Cmp(e36) < 0 {
		// B^2 cannot be calculated directly at 1e18 scale without overflowing
		tmp.Mul(&absB, &absB)
		tmp.Add(&tmp, fourAC)
		sqrt, err = _Sqrt(&tmp, true)
		if err != nil {
			return nil, err
		}
	} else {
		// B^2 can be calculated directly at 1e18 scale without overflowing
		scale := computeScale(&absB)

		tmp.Div(&absB, scale)
		squaredB, err := v3Utils.MulDiv(&tmp, &absB, scale)
		if err != nil {
			return nil, err
		}

		tmp.Mul(scale, scale)
		term2.Div(fourAC, &tmp)

		tmp.Add(squaredB, &term2)
		sqrt, err = _Sqrt(&tmp, true)
		if err != nil {
			return nil, err
		}
		sqrt.Mul(sqrt, scale)
	}

	var x *uint256.Int
	if B.Sign() <= 0 {
		// use the regular quadratic formula solution (-b + sqrt(b^2 - 4ac)) / 2a
		tmp.Add(&absB, sqrt)
		term2.Mul(c, big256.U2)

		x, err = v3Utils.MulDivRoundingUp(&tmp, big256.BONE, &term2)
		if err != nil {
			return nil, err
		}
	} else {
		// use the "citardauq" quadratic formula solution 2c / (-b - sqrt(b^2 - 4ac))
		tmp.Add(&absB, sqrt)
		term2.Mul(C, big256.U2)

		var err error
		x, err = _CeilDiv(&term2, &tmp)
		if err != nil {
			return nil, err
		}
	}

	x.AddUint64(x, 1)

	if x.Cmp(x0) >= 0 {
		return new(uint256.Int).Set(x0), nil
	}

	return x, nil
}

func computeScale(x *uint256.Int) *uint256.Int {
	bits := x.BitLen()

	if bits > 128 {
		excessBits := bits - 128
		var result uint256.Int
		result.Lsh(big256.U1, uint(excessBits))
		return &result
	}
	return big256.U1
}

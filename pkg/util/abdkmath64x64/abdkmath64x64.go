// Package abdkmath64x64 is a wei-exact Go port of the signed 64.64-bit fixed-point
// math library ABDKMath64x64 (https://github.com/abdk-consulting/abdk-libraries-solidity).
//
// A signed 64.64-bit fixed-point number is a fraction whose numerator is a signed
// 128-bit integer and whose denominator is 2^64. As in the Solidity original, values
// are represented by the numerator only. Here that numerator is carried in a
// signed 256-bit integer (github.com/KyberNetwork/int256), always constrained to the
// int128 range for valid values, mirroring how the Solidity library casts int128 to
// int256 for intermediate computation.
//
// The port is bit-for-bit faithful to
// ../lmsr-amm/lib/abdk-libraries-solidity/ABDKMath64x64.sol: every operation reproduces
// the exact rounding (truncation toward zero for division/mul, arithmetic shifts, the
// unsigned 512-bit helpers in mulu/divuu, and the exp_2/log_2 magic-constant ladders).
// Solidity `require`s become returned errors so callers can reject reverting swaps
// instead of panicking.
//
// The library is intentionally standalone and reusable — it does not depend on any
// liquidity-source package — so other DEX integrations backed by ABDKMath64x64 can
// reuse it. It is backed by holiman/uint256 and KyberNetwork/int256 for performance
// (stack-friendly value types, no big.Int) as recommended for hot pricing paths.
package abdkmath64x64

import (
	"errors"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

// Sentinel errors mirror the ABDKMath64x64 `require` failures.
var (
	// ErrOverflow is returned when a result would fall outside the int128 (64.64) range,
	// or when an unsigned helper result exceeds its declared width. Maps to the "too large"
	// class of pool reverts.
	ErrOverflow = errors.New("abdk: overflow")
	// ErrDivByZero is returned when dividing by zero.
	ErrDivByZero = errors.New("abdk: division by zero")
	// ErrNegative is returned by MulU when the 64.64 operand is negative (Solidity require(x >= 0)).
	ErrNegative = errors.New("abdk: negative operand")
	// ErrNonPositive is returned by Ln/Log2 when x <= 0 (Solidity require(x > 0)).
	ErrNonPositive = errors.New("abdk: non-positive logarithm argument")
)

// hexI parses a non-negative hex literal (< 2^255) into a signed int256 value.
func hexI(s string) *int256.Int {
	u := uint256.MustFromHex(s)
	r := asI(u)
	return &r
}

// hexU parses a hex literal into an unsigned uint256 value.
func hexU(s string) *uint256.Int {
	return uint256.MustFromHex(s)
}

var (
	// ONE is 1.0 in 64.64 fixed point (2^64), matching LMSRKernel.ONE.
	ONE = hexI("0x10000000000000000")
	// EXP_LIMIT is 32.0 in 64.64 fixed point (0x200000000000000000), matching LMSRKernel.EXP_LIMIT.
	// The kernel uses it as a swap-size guard; it is not enforced inside ABDK exp itself.
	EXP_LIMIT = hexI("0x200000000000000000")

	// min64x64 = -2^127, max64x64 = 2^127 - 1: the representable int128 range.
	min64x64 = new(int256.Int).Neg(hexI("0x80000000000000000000000000000000"))
	max64x64 = hexI("0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")

	// Unsigned range/mask constants used by the unsigned helpers.
	max64x64u = hexU("0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")                 // uint128(MAX_64x64) = 2^127 - 1
	max128u   = hexU("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")                 // 2^128 - 1
	max192u   = hexU("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF") // 2^192 - 1
	mask128   = hexU("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")                 // low 128 bits
	uintOne   = uint256.NewInt(1)
)

// asU reinterprets a signed int256 as its raw two's-complement 256-bit pattern (uint256).
// int256.Int and uint256.Int are both little-endian [4]uint64, so this is a bit-exact copy
// — precisely Solidity's uint256(int256(x)) cast.
func asU(x *int256.Int) uint256.Int {
	return uint256.Int{x[0], x[1], x[2], x[3]}
}

// asI reinterprets a raw 256-bit pattern (uint256) as a signed int256 — Solidity's
// int256(uint256(...)) cast.
func asI(x *uint256.Int) int256.Int {
	return int256.Int{x[0], x[1], x[2], x[3]}
}

// toInt128 truncates a raw 256-bit pattern to its low 128 bits and sign-extends from
// bit 127 — Solidity's `int128(int256(...))` cast, which discards the high 128 bits and
// treats bit 127 as the sign. Needed where a computation intentionally overflows and
// relies on the narrowing cast to recover a signed value (e.g. ABDKMath64x64.ln).
func toInt128(x *uint256.Int) int256.Int {
	low := int256.Int{x[0], x[1], 0, 0}
	if x[1]>>63 == 1 { // bit 127 set -> negative: sign-extend the high 128 bits
		low[2] = ^uint64(0)
		low[3] = ^uint64(0)
	}
	return low
}

// inRange reports whether r is within [min64x64, max64x64].
func inRange(r *int256.Int) bool {
	return r.Cmp(min64x64) >= 0 && r.Cmp(max64x64) <= 0
}

// Add computes x + y, reverting on int128 overflow. Mirrors ABDKMath64x64.add.
func Add(x, y *int256.Int) (int256.Int, error) {
	var r int256.Int
	r.Add(x, y) // exact: |x|,|y| < 2^127 so the sum fits well within int256
	if !inRange(&r) {
		return int256.Int{}, ErrOverflow
	}
	return r, nil
}

// Sub computes x - y, reverting on int128 overflow. Mirrors ABDKMath64x64.sub.
func Sub(x, y *int256.Int) (int256.Int, error) {
	var r int256.Int
	r.Sub(x, y)
	if !inRange(&r) {
		return int256.Int{}, ErrOverflow
	}
	return r, nil
}

// Mul computes x * y rounding toward negative infinity (arithmetic shift), reverting on
// int128 overflow. Mirrors ABDKMath64x64.mul: int256(x) * y >> 64.
func Mul(x, y *int256.Int) (int256.Int, error) {
	var p, r int256.Int
	p.Mul(x, y)   // exact: product magnitude < 2^254
	r.Rsh(&p, 64) // arithmetic shift right (floors toward -inf, as Solidity `>>` on signed)
	if !inRange(&r) {
		return int256.Int{}, ErrOverflow
	}
	return r, nil
}

// Div computes x / y rounding toward zero, reverting on zero divisor or int128 overflow.
// Mirrors ABDKMath64x64.div: (int256(x) << 64) / y.
func Div(x, y *int256.Int) (int256.Int, error) {
	if y.IsZero() {
		return int256.Int{}, ErrDivByZero
	}
	var sh, q int256.Int
	sh.Lsh(x, 64) // exact: |x| < 2^127 so |x<<64| < 2^191
	q.Quo(&sh, y) // truncated (toward zero) division, matching Solidity signed `/`
	if !inRange(&q) {
		return int256.Int{}, ErrOverflow
	}
	return q, nil
}

// Neg computes -x, reverting only for x == MIN_64x64. Mirrors ABDKMath64x64.neg.
func Neg(x *int256.Int) (int256.Int, error) {
	if x.Cmp(min64x64) == 0 {
		return int256.Int{}, ErrOverflow
	}
	var r int256.Int
	r.Neg(x)
	return r, nil
}

// DivU computes x / y as a 64.64 fixed-point number, where x and y are unsigned 256-bit
// integers, rounding toward zero. Reverts on zero divisor or when the result exceeds
// the 64.64 range. Mirrors ABDKMath64x64.divu.
func DivU(x, y *uint256.Int) (int256.Int, error) {
	if y.IsZero() {
		return int256.Int{}, ErrDivByZero
	}
	r, err := divuu(x, y)
	if err != nil {
		return int256.Int{}, err
	}
	if r.Cmp(max64x64u) > 0 {
		return int256.Int{}, ErrOverflow
	}
	res := asI(&r)
	return res, nil
}

// MulU computes x * y rounding down, where x is a 64.64 fixed-point number (must be >= 0)
// and y is an unsigned 256-bit integer, returning an unsigned 256-bit integer. Reverts on
// negative x or on overflow. Mirrors ABDKMath64x64.mulu.
func MulU(x *int256.Int, y *uint256.Int) (uint256.Int, error) {
	if y.IsZero() {
		return uint256.Int{}, nil
	}
	if x.Sign() < 0 {
		return uint256.Int{}, ErrNegative
	}
	xu := asU(x) // x >= 0, so this equals x

	var ylo, prod, lo uint256.Int
	ylo.And(y, mask128)
	prod.Mul(&xu, &ylo)
	lo.Rsh(&prod, 64)

	var yhi, hi uint256.Int
	yhi.Rsh(y, 128)
	hi.Mul(&xu, &yhi)
	if hi.Cmp(max192u) > 0 {
		return uint256.Int{}, ErrOverflow
	}

	var hiShifted, sum uint256.Int
	hiShifted.Lsh(&hi, 64)
	if _, over := sum.AddOverflow(&hiShifted, &lo); over {
		return uint256.Int{}, ErrOverflow
	}
	return sum, nil
}

// divuu computes x / y as an unsigned 64.64 fixed-point value (numerator only), where x and
// y are unsigned 256-bit integers, rounding toward zero. Mirrors the private
// ABDKMath64x64.divuu bit-for-bit, including the 512-bit long-division correction step.
func divuu(x, y *uint256.Int) (uint256.Int, error) {
	// y != 0 already guaranteed by callers, but keep the guard for standalone use.
	if y.IsZero() {
		return uint256.Int{}, ErrDivByZero
	}

	var result uint256.Int

	if x.Cmp(max192u) <= 0 {
		// Fast path: (x << 64) / y fits without overflow.
		var xsh uint256.Int
		xsh.Lsh(x, 64)
		result.Div(&xsh, y)
	} else {
		// Normalize x to its most-significant bit, then do a corrected long division.
		msb := uint(192)
		var xc uint256.Int
		xc.Rsh(x, 192)
		if xc.Cmp(hexU("0x100000000")) >= 0 {
			xc.Rsh(&xc, 32)
			msb += 32
		}
		if xc.Cmp(hexU("0x10000")) >= 0 {
			xc.Rsh(&xc, 16)
			msb += 16
		}
		if xc.Cmp(hexU("0x100")) >= 0 {
			xc.Rsh(&xc, 8)
			msb += 8
		}
		if xc.Cmp(hexU("0x10")) >= 0 {
			xc.Rsh(&xc, 4)
			msb += 4
		}
		if xc.Cmp(hexU("0x4")) >= 0 {
			xc.Rsh(&xc, 2)
			msb += 2
		}
		if xc.Cmp(hexU("0x2")) >= 0 {
			msb++
		}

		// result = (x << (255 - msb)) / (((y - 1) >> (msb - 191)) + 1)
		var xShift, ym1, divisor uint256.Int
		xShift.Lsh(x, 255-msb)
		ym1.Sub(y, uintOne)
		divisor.Rsh(&ym1, msb-191)
		divisor.Add(&divisor, uintOne)
		result.Div(&xShift, &divisor)
		if result.Cmp(max128u) > 0 {
			return uint256.Int{}, ErrOverflow
		}

		// Correction: recompute the remainder over the full 256-bit dividend.
		var yhi, ylo, hi, lo uint256.Int
		yhi.Rsh(y, 128)
		ylo.And(y, mask128)
		hi.Mul(&result, &yhi)
		lo.Mul(&result, &ylo)

		var xh, xl uint256.Int
		xh.Rsh(x, 192)
		xl.Lsh(x, 64)

		if xl.Cmp(&lo) < 0 {
			xh.Sub(&xh, uintOne)
		}
		xl.Sub(&xl, &lo)
		lo.Lsh(&hi, 128)
		if xl.Cmp(&lo) < 0 {
			xh.Sub(&xh, uintOne)
		}
		xl.Sub(&xl, &lo)

		var hiShift uint256.Int
		hiShift.Rsh(&hi, 128)
		if xh.Cmp(&hiShift) == 0 {
			var q uint256.Int
			q.Div(&xl, y)
			result.Add(&result, &q)
		} else {
			result.Add(&result, uintOne)
		}
	}

	if result.Cmp(max128u) > 0 {
		return uint256.Int{}, ErrOverflow
	}
	return result, nil
}

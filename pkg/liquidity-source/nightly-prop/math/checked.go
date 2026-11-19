// Package math implements the pool's integer pricing and rounding rules.
package math

import "github.com/holiman/uint256"

var one = uint256.NewInt(1)

// Rounding is OpenZeppelin's Math.Rounding, limited to the two modes that the pool uses.
type Rounding uint8

const (
	Floor Rounding = iota
	Ceil
)

// Add returns a + b. A sum above uint256 is ErrOverflow, as Solidity's checked addition reverts.
func Add(a, b *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if _, overflow := z.AddOverflow(a, b); overflow {
		return z, ErrOverflow
	}
	return z, nil
}

// Sub returns a - b. A negative result is ErrOverflow, as Solidity's checked subtraction reverts.
func Sub(a, b *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if _, underflow := z.SubOverflow(a, b); underflow {
		return z, ErrOverflow
	}
	return z, nil
}

// Mul returns a * b. A product above uint256 is ErrOverflow, as Solidity's checked multiplication
// reverts.
func Mul(a, b *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if _, overflow := z.MulOverflow(a, b); overflow {
		return z, ErrOverflow
	}
	return z, nil
}

// Min is OpenZeppelin's Math.min.
func Min(a, b *uint256.Int) uint256.Int {
	if a.Lt(b) {
		return *a
	}
	return *b
}

// MulDiv is OpenZeppelin's Math.mulDiv(x, y, denominator, rounding): x * y / denominator with the
// full 512-bit product. A zero denominator is ErrDivisionByZero. A quotient above uint256, also
// after rounding up, is ErrOverflow.
func MulDiv(x, y, denominator *uint256.Int, rounding Rounding) (uint256.Int, error) {
	var z uint256.Int
	// MulDivOverflow and MulMod return zero for a zero denominator, where OpenZeppelin reverts.
	if denominator.IsZero() {
		return z, ErrDivisionByZero
	}
	if _, overflow := z.MulDivOverflow(x, y, denominator); overflow {
		return z, ErrOverflow
	}
	if rounding == Ceil {
		var remainder uint256.Int
		if !remainder.MulMod(x, y, denominator).IsZero() {
			return Add(&z, one)
		}
	}
	return z, nil
}

// CeilDiv is OpenZeppelin's Math.ceilDiv: a / b rounded up. A zero b is ErrDivisionByZero.
func CeilDiv(a, b *uint256.Int) (uint256.Int, error) {
	var z uint256.Int
	if b.IsZero() {
		return z, ErrDivisionByZero
	}
	if a.IsZero() {
		return z, nil
	}
	z.Sub(a, one)
	z.Div(&z, b)
	z.AddUint64(&z, 1)
	return z, nil
}

// Sqrt is OpenZeppelin's Math.sqrt(a, rounding): the square root of a, rounded in the direction of
// rounding. uint256's Sqrt gives the exact floor, as OpenZeppelin's algorithm does.
func Sqrt(a *uint256.Int, rounding Rounding) uint256.Int {
	var z, square uint256.Int
	z.Sqrt(a)
	// z is below 2^128, so its square fits uint256.
	if rounding == Ceil && square.Mul(&z, &z).Lt(a) {
		z.AddUint64(&z, 1)
	}
	return z
}

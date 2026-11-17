// Package math ports the math of the NightlyPool contract (evm_contracts/src/lib in the Nightly
// trading engine) to Go, with the same integer results. Where the contract reverts, a function
// here returns an error.
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
func Add(a, b *uint256.Int) (*uint256.Int, error) {
	z, overflow := new(uint256.Int).AddOverflow(a, b)
	if overflow {
		return nil, ErrOverflow
	}
	return z, nil
}

// Sub returns a - b. A negative result is ErrOverflow, as Solidity's checked subtraction reverts.
func Sub(a, b *uint256.Int) (*uint256.Int, error) {
	z, underflow := new(uint256.Int).SubOverflow(a, b)
	if underflow {
		return nil, ErrOverflow
	}
	return z, nil
}

// Mul returns a * b. A product above uint256 is ErrOverflow, as Solidity's checked multiplication
// reverts.
func Mul(a, b *uint256.Int) (*uint256.Int, error) {
	z, overflow := new(uint256.Int).MulOverflow(a, b)
	if overflow {
		return nil, ErrOverflow
	}
	return z, nil
}

// Min is OpenZeppelin's Math.min.
func Min(a, b *uint256.Int) *uint256.Int {
	if a.Lt(b) {
		return new(uint256.Int).Set(a)
	}
	return new(uint256.Int).Set(b)
}

// MulDiv is OpenZeppelin's Math.mulDiv(x, y, denominator, rounding): x * y / denominator with the
// full 512-bit product. A zero denominator is ErrDivisionByZero. A quotient above uint256, also
// after rounding up, is ErrOverflow.
func MulDiv(x, y, denominator *uint256.Int, rounding Rounding) (*uint256.Int, error) {
	// MulDivOverflow and MulMod return zero for a zero denominator, where OpenZeppelin reverts.
	if denominator.IsZero() {
		return nil, ErrDivisionByZero
	}
	z, overflow := new(uint256.Int).MulDivOverflow(x, y, denominator)
	if overflow {
		return nil, ErrOverflow
	}
	if rounding == Ceil && !new(uint256.Int).MulMod(x, y, denominator).IsZero() {
		return Add(z, one)
	}
	return z, nil
}

// CeilDiv is OpenZeppelin's Math.ceilDiv: a / b rounded up. A zero b is ErrDivisionByZero.
func CeilDiv(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, ErrDivisionByZero
	}
	if a.IsZero() {
		return new(uint256.Int), nil
	}
	z := new(uint256.Int).Sub(a, one)
	z.Div(z, b)
	return z.AddUint64(z, 1), nil
}

// Sqrt is OpenZeppelin's Math.sqrt(a, rounding): the square root of a, rounded in the direction of
// rounding. uint256's Sqrt gives the exact floor, as OpenZeppelin's algorithm does.
func Sqrt(a *uint256.Int, rounding Rounding) *uint256.Int {
	z := new(uint256.Int).Sqrt(a)
	// z is below 2^128, so its square fits uint256.
	if rounding == Ceil && new(uint256.Int).Mul(z, z).Lt(a) {
		z.AddUint64(z, 1)
	}
	return z
}

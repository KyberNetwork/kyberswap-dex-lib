// Package ilyris implements Ilyris bin-AMM pricing for KyberSwap's dex-lib.
//
// This is a THIRD implementation of the same maths: Solidity (BinPool.sol) is the source of
// truth, TypeScript (sdk/typescript/src/pool.ts) is the second, and this is the third.
// KyberSwap simulates swaps off-chain in Go, so a route it quotes and a swap that executes
// only agree if this file reproduces the contract exactly — not approximately.
//
// The parity oracle in ../../sdk/typescript/test/oracle.json is generated from the real
// BinPool on an in-process EVM, so parity_test.go checks this code against the CONTRACT
// rather than against the TypeScript port. Two ports agreeing with each other proves nothing
// if they share a misreading.
//
// # Why every division is spelled out
//
// Solidity integer division truncates toward zero, and every truncation here is load-bearing:
// the per-step floor inside powX18 is part of the on-chain price, not an artefact of it.
// Replacing it with a single high-precision pow gives a different, wrong, price. Likewise the
// floor/ceil split between quoteFromX and quoteFromXUp is what stops a swap paying out more
// than a bin holds. Every helper below names its rounding direction, and callers must pick
// deliberately.
//
// All values are non-negative, so big.Int's Div (Euclidean) and Quo (truncated) agree; Div is
// used throughout for clarity.
package ilyris

import (
	"errors"
	"fmt"
	"math/big"
)

// Scaling constants. These MUST match BinMath.sol and the TypeScript port.
var (
	// Scale is 1e18, the fixed-point base for prices.
	Scale = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	// BPS is 10_000, basis points.
	BPS = big.NewInt(10_000)
	// FeePrecision is 1e9, the precision fee rates are expressed at.
	FeePrecision = big.NewInt(1_000_000_000)
	// MaxFeeRate caps the total fee at 10% (1e8 at 1e9 precision).
	MaxFeeRate = big.NewInt(100_000_000)
	// variableFeeScale is 1e11, the divisor in the dynamic-fee surcharge.
	variableFeeScale = big.NewInt(100_000_000_000)
	// binStepScale is 1e14; binStepBps * 1e14 is the per-bin ratio increment.
	binStepScale = big.NewInt(100_000_000_000_000)

	// uint256Max is (1<<256)-1. Solidity reverts past it; so do we, rather than letting
	// big.Int silently grow and produce a number the chain could never return.
	uint256Max = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

	one = big.NewInt(1)
)

// Gas hints, measured in test/GasProfile.t.sol. Must stay identical to BinPoolLens.sol
// and services/indexer/quote.mjs — verify.ps1 fails the three-way check if they drift.
// A SWAP crossing bins: BASE + (binsCrossed-1)*PER_EXTRA. Not deposit gas.
const (
	BaseSwapGas    = 130_000
	PerExtraBinGas = 41_500
)

const (
	// MinBinID and MaxBinID bound the addressable book.
	MinBinID = -500_000
	MaxBinID = 500_000
)

// ErrInsufficientLiquidity is returned when the book cannot fill a requested size. It is a
// distinct error rather than a zero quote on purpose: an aggregator that reads zero as a
// price will route into it, and the swap then reverts with the user paying the gas.
var ErrInsufficientLiquidity = errors.New("ilyris: insufficient liquidity")

// ErrOverflow mirrors a Solidity revert on a value that cannot fit in uint256.
var ErrOverflow = errors.New("ilyris: uint256 overflow")

func requireUint256(v *big.Int, name string) error {
	if v.Sign() < 0 {
		return fmt.Errorf("%w: %s is negative", ErrOverflow, name)
	}
	if v.Cmp(uint256Max) > 0 {
		return fmt.Errorf("%w: %s exceeds uint256", ErrOverflow, name)
	}
	return nil
}

// MulDiv computes floor(x*y/denominator) with full intermediate precision.
//
// Solidity's 512-bit mulDiv reverts when the true quotient will not fit in uint256. big.Int
// would happily return it, so the bound is checked explicitly — otherwise this would quote
// sizes the contract refuses.
func MulDiv(x, y, denominator *big.Int) (*big.Int, error) {
	if err := requireUint256(x, "mulDiv x"); err != nil {
		return nil, err
	}
	if err := requireUint256(y, "mulDiv y"); err != nil {
		return nil, err
	}
	if denominator.Sign() <= 0 {
		return nil, errors.New("ilyris: mulDiv division by zero")
	}
	result := new(big.Int).Mul(x, y)
	result.Div(result, denominator)
	if err := requireUint256(result, "mulDiv result"); err != nil {
		return nil, err
	}
	return result, nil
}

// MulDivUp computes ceil(x*y/denominator).
func MulDivUp(x, y, denominator *big.Int) (*big.Int, error) {
	if err := requireUint256(x, "mulDivUp x"); err != nil {
		return nil, err
	}
	if err := requireUint256(y, "mulDivUp y"); err != nil {
		return nil, err
	}
	if denominator.Sign() <= 0 {
		return nil, errors.New("ilyris: mulDivUp division by zero")
	}
	product := new(big.Int).Mul(x, y)
	q, rem := new(big.Int).QuoRem(product, denominator, new(big.Int))
	if rem.Sign() != 0 {
		q.Add(q, one)
	}
	if err := requireUint256(q, "mulDivUp result"); err != nil {
		return nil, err
	}
	return q, nil
}

// powX18 is fixed-point exponentiation by squaring, flooring at EVERY step.
//
// The per-step floor is part of the on-chain price. Computing base^n at high precision and
// rounding once gives a different answer, and the difference compounds with |id| — so a pool
// far from bin 0 would be priced wrongly while bins near 0 looked fine.
func powX18(base, n *big.Int) (*big.Int, error) {
	z := new(big.Int).Set(Scale)
	x := new(big.Int).Set(base)
	e := new(big.Int).Set(n)

	for e.Sign() != 0 {
		if e.Bit(0) == 1 {
			var err error
			z, err = MulDiv(z, x, Scale)
			if err != nil {
				return nil, err
			}
		}
		e.Rsh(e, 1)
		if e.Sign() != 0 {
			var err error
			x, err = MulDiv(x, x, Scale)
			if err != nil {
				return nil, err
			}
		}
	}
	return z, nil
}

// PriceFromID returns the human quote-per-base price for a bin, scaled by 1e18.
func PriceFromID(binStepBps int, id int) (*big.Int, error) {
	if binStepBps <= 0 || binStepBps > 1000 {
		return nil, fmt.Errorf("ilyris: invalid bin step %d", binStepBps)
	}
	if id < MinBinID || id > MaxBinID {
		return nil, fmt.Errorf("ilyris: bin id %d out of range", id)
	}

	baseX18 := new(big.Int).Add(Scale, new(big.Int).Mul(big.NewInt(int64(binStepBps)), binStepScale))

	absID := id
	if absID < 0 {
		absID = -absID
	}
	ratioX18, err := powX18(baseX18, big.NewInt(int64(absID)))
	if err != nil {
		return nil, err
	}

	var priceX18 *big.Int
	if id >= 0 {
		priceX18 = ratioX18
	} else {
		// Negative ids invert, and the inversion is a separate floor. Computing
		// base^(-n) directly would round differently.
		priceX18, err = MulDiv(Scale, Scale, ratioX18)
		if err != nil {
			return nil, err
		}
	}
	if priceX18.Sign() <= 0 {
		return nil, errors.New("ilyris: bin price underflow")
	}
	return priceX18, nil
}

func pow10(exp int) (*big.Int, error) {
	if exp < 0 || exp > 18 {
		return nil, fmt.Errorf("ilyris: invalid decimal scale %d", exp)
	}
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil), nil
}

// decimalDivisor builds the denominator used converting X->Y, and the numerator used
// converting Y->X. Split out because getting the branch backwards is silent: it produces a
// price wrong by a power of ten, which looks like a units bug rather than a maths bug.
func decimalDivisor(decimalsX, decimalsY int) (*big.Int, error) {
	if decimalsX < 0 || decimalsX > 18 || decimalsY < 0 || decimalsY > 18 {
		return nil, fmt.Errorf("ilyris: invalid decimals %d/%d", decimalsX, decimalsY)
	}
	if decimalsX >= decimalsY {
		p, err := pow10(decimalsX - decimalsY)
		if err != nil {
			return nil, err
		}
		return new(big.Int).Mul(Scale, p), nil
	}
	p, err := pow10(decimalsY - decimalsX)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Div(Scale, p), nil
}

// QuoteFromX converts raw X to raw Y at priceX18, rounding DOWN.
func QuoteFromX(amountX, priceX18 *big.Int, decimalsX, decimalsY int) (*big.Int, error) {
	d, err := decimalDivisor(decimalsX, decimalsY)
	if err != nil {
		return nil, err
	}
	return MulDiv(amountX, priceX18, d)
}

// QuoteFromXUp converts raw X to raw Y at priceX18, rounding UP.
func QuoteFromXUp(amountX, priceX18 *big.Int, decimalsX, decimalsY int) (*big.Int, error) {
	d, err := decimalDivisor(decimalsX, decimalsY)
	if err != nil {
		return nil, err
	}
	return MulDivUp(amountX, priceX18, d)
}

// XFromQuote converts raw Y to raw X at priceX18, rounding DOWN.
func XFromQuote(amountY, priceX18 *big.Int, decimalsX, decimalsY int) (*big.Int, error) {
	d, err := decimalDivisor(decimalsX, decimalsY)
	if err != nil {
		return nil, err
	}
	return MulDiv(amountY, d, priceX18)
}

// XFromQuoteUp converts raw Y to raw X at priceX18, rounding UP.
func XFromQuoteUp(amountY, priceX18 *big.Int, decimalsX, decimalsY int) (*big.Int, error) {
	d, err := decimalDivisor(decimalsX, decimalsY)
	if err != nil {
		return nil, err
	}
	return MulDivUp(amountY, d, priceX18)
}

// ceilDiv computes ceil(a/b) for non-negative a and positive b.
func ceilDiv(a, b *big.Int) *big.Int {
	q, rem := new(big.Int).QuoRem(a, b, new(big.Int))
	if rem.Sign() != 0 {
		q.Add(q, one)
	}
	return q
}

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
// # Why uint256.Int and not big.Int
//
// Per AGENTS.md, math and state use uint256.Int. Beyond the allocation win, it makes this file
// a closer mirror of the contract than big.Int was. Solidity's uint256 cannot be negative and
// cannot exceed 2^256-1; big.Int can do both, so the previous version carried an explicit
// requireUint256 guard on every operand and every result to re-impose a bound the EVM gets for
// free. Those guards are gone: the type is the bound. What remains is the one thing the type
// does NOT give us, which is that a mulDiv whose true quotient overflows must revert rather
// than wrap, and that is read from MulDivOverflow's carry flag below.
package ilyris

import (
	"errors"
	"fmt"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Scaling constants. These MUST match BinMath.sol and the TypeScript port.
//
// Read-only. Nothing in this package uses one as a receiver, which matters more for pointers
// to shared uint256.Int values than it did for big.Int: a single stray Add would corrupt the
// constant for every goroutine quoting concurrently.
var (
	// Scale is 1e18, the fixed-point base for prices.
	Scale = big256.TenPow(18)
	// BPS is 10_000, basis points.
	BPS = big256.NewI(10_000)
	// FeePrecision is 1e9, the precision fee rates are expressed at.
	FeePrecision = big256.TenPow(9)
	// MaxFeeRate caps the total fee at 10% (1e8 at 1e9 precision).
	MaxFeeRate = big256.TenPow(8)
	// variableFeeScale is 1e11, the divisor in the dynamic-fee surcharge.
	variableFeeScale = big256.TenPow(11)
	// binStepScale is 1e14; binStepBps * 1e14 is the per-bin ratio increment.
	binStepScale = big256.TenPow(14)
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

// ErrDivByZero mirrors a Solidity division by zero.
var ErrDivByZero = errors.New("ilyris: division by zero")

// MulDiv computes floor(x*y/denominator) with full 512-bit intermediate precision.
//
// Solidity's mulDiv reverts when the true quotient will not fit in uint256. MulDivOverflow
// computes the same 512-bit product and reports exactly that condition in its second return,
// so the check is the contract's check rather than a bound re-imposed afterwards.
//
// NOTE: deliberately not big256.MulDiv, which calls the same primitive but discards the
// overflow flag. Silently truncating here would quote a size the pool would refuse.
func MulDiv(x, y, denominator *uint256.Int) (*uint256.Int, error) {
	if denominator.IsZero() {
		return nil, fmt.Errorf("%w: mulDiv", ErrDivByZero)
	}
	var res uint256.Int
	if _, overflow := res.MulDivOverflow(x, y, denominator); overflow {
		return nil, fmt.Errorf("%w: mulDiv result", ErrOverflow)
	}
	return &res, nil
}

// MulDivUp computes ceil(x*y/denominator).
//
// Same reasoning as MulDiv: big256.MulDivUp wraps this call and drops the error, and the
// error is the whole point.
func MulDivUp(x, y, denominator *uint256.Int) (*uint256.Int, error) {
	if denominator.IsZero() {
		return nil, fmt.Errorf("%w: mulDivUp", ErrDivByZero)
	}
	var res uint256.Int
	if err := v3Utils.MulDivRoundingUpV2(x, y, denominator, &res); err != nil {
		return nil, fmt.Errorf("%w: mulDivUp result", ErrOverflow)
	}
	return &res, nil
}

// powX18 is fixed-point exponentiation by squaring, flooring at EVERY step.
//
// The per-step floor is part of the on-chain price. Computing base^n at high precision and
// rounding once gives a different answer, and the difference compounds with |id| — so a pool
// far from bin 0 would be priced wrongly while bins near 0 looked fine.
func powX18(base *uint256.Int, n uint64) (*uint256.Int, error) {
	z := new(uint256.Int).Set(Scale)
	x := new(uint256.Int).Set(base)

	for e := n; e != 0; e >>= 1 {
		if e&1 == 1 {
			var err error
			if z, err = MulDiv(z, x, Scale); err != nil {
				return nil, err
			}
		}
		if e>>1 != 0 {
			var err error
			if x, err = MulDiv(x, x, Scale); err != nil {
				return nil, err
			}
		}
	}
	return z, nil
}

// PriceFromID returns the human quote-per-base price for a bin, scaled by 1e18.
func PriceFromID(binStepBps int, id int) (*uint256.Int, error) {
	if binStepBps <= 0 || binStepBps > 1000 {
		return nil, fmt.Errorf("ilyris: invalid bin step %d", binStepBps)
	}
	if id < MinBinID || id > MaxBinID {
		return nil, fmt.Errorf("ilyris: bin id %d out of range", id)
	}

	var step, baseX18 uint256.Int
	step.SetUint64(uint64(binStepBps))
	baseX18.Add(Scale, step.Mul(&step, binStepScale))

	absID := id
	if absID < 0 {
		absID = -absID
	}
	ratioX18, err := powX18(&baseX18, uint64(absID))
	if err != nil {
		return nil, err
	}

	priceX18 := ratioX18
	if id < 0 {
		// Negative ids invert, and the inversion is a separate floor. Computing
		// base^(-n) directly would round differently.
		if priceX18, err = MulDiv(Scale, Scale, ratioX18); err != nil {
			return nil, err
		}
	}
	if priceX18.IsZero() {
		return nil, errors.New("ilyris: bin price underflow")
	}
	return priceX18, nil
}

func pow10(exp int) (*uint256.Int, error) {
	if exp < 0 || exp > 18 {
		return nil, fmt.Errorf("ilyris: invalid decimal scale %d", exp)
	}
	return big256.TenPow(exp), nil
}

// decimalDivisor builds the denominator used converting X->Y, and the numerator used
// converting Y->X. Split out because getting the branch backwards is silent: it produces a
// price wrong by a power of ten, which looks like a units bug rather than a maths bug.
func decimalDivisor(decimalsX, decimalsY int) (*uint256.Int, error) {
	if decimalsX < 0 || decimalsX > 18 || decimalsY < 0 || decimalsY > 18 {
		return nil, fmt.Errorf("ilyris: invalid decimals %d/%d", decimalsX, decimalsY)
	}
	if decimalsX >= decimalsY {
		p, err := pow10(decimalsX - decimalsY)
		if err != nil {
			return nil, err
		}
		// Bounded by 1e18 * 1e18, so this cannot overflow uint256; Mul is safe unchecked.
		return new(uint256.Int).Mul(Scale, p), nil
	}
	p, err := pow10(decimalsY - decimalsX)
	if err != nil {
		return nil, err
	}
	return new(uint256.Int).Div(Scale, p), nil
}

// QuoteFromX converts raw X to raw Y at priceX18, rounding DOWN.
func QuoteFromX(amountX, priceX18 *uint256.Int, decimalsX, decimalsY int) (*uint256.Int, error) {
	d, err := decimalDivisor(decimalsX, decimalsY)
	if err != nil {
		return nil, err
	}
	return MulDiv(amountX, priceX18, d)
}

// QuoteFromXUp converts raw X to raw Y at priceX18, rounding UP.
func QuoteFromXUp(amountX, priceX18 *uint256.Int, decimalsX, decimalsY int) (*uint256.Int, error) {
	d, err := decimalDivisor(decimalsX, decimalsY)
	if err != nil {
		return nil, err
	}
	return MulDivUp(amountX, priceX18, d)
}

// XFromQuote converts raw Y to raw X at priceX18, rounding DOWN.
func XFromQuote(amountY, priceX18 *uint256.Int, decimalsX, decimalsY int) (*uint256.Int, error) {
	d, err := decimalDivisor(decimalsX, decimalsY)
	if err != nil {
		return nil, err
	}
	return MulDiv(amountY, d, priceX18)
}

// XFromQuoteUp converts raw Y to raw X at priceX18, rounding UP.
func XFromQuoteUp(amountY, priceX18 *uint256.Int, decimalsX, decimalsY int) (*uint256.Int, error) {
	d, err := decimalDivisor(decimalsX, decimalsY)
	if err != nil {
		return nil, err
	}
	return MulDivUp(amountY, d, priceX18)
}

// ceilDiv computes ceil(a/b) for positive b. Zero b is the caller's bug, and returning zero
// would silently under-quote, so it is reported.
func ceilDiv(a, b *uint256.Int) (*uint256.Int, error) {
	if b.IsZero() {
		return nil, fmt.Errorf("%w: ceilDiv", ErrDivByZero)
	}
	return big256.DivUp(a, b), nil
}

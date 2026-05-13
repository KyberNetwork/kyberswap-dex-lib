package lunarbase

import (
	"math"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Direct port of the on-chain SwapLib (fix/incident, single-price Q32.48
// design). Bit-for-bit identical with `math/rust/lunarbase-pmm-math` and
// `math/go/lunarbasepmm`. Vectors in testdata/ are shared.

const fixedPoint48Resolution = 48

var (
	q12     = new(uint256.Int).Lsh(big256.U1, 12)
	q24     = new(uint256.Int).Lsh(big256.U1, 24)
	q48     = new(uint256.Int).Lsh(big256.U1, 48)
	u2Pow80 = new(uint256.Int).Lsh(big256.U1, 80)
)

func isqrt(x *uint256.Int) *uint256.Int {
	return new(uint256.Int).Sqrt(x)
}

func ceilDiv(dst, a, b *uint256.Int) *uint256.Int {
	var rem uint256.Int
	dst.DivMod(a, b, &rem)
	if !rem.IsZero() {
		dst.AddUint64(dst, 1)
	}
	return dst
}

// concentrationQ48 writes c = mulDiv(concentrationK, r², Q12) into dst, where
// r is wealth-normalised by cascading mulDiv(_, sqrtP_Q48, Q48) twice to
// compute reserveX * P precisely. Saturates at Q48 (100%).
//
// Returns dst zeroed when amountIn, k, or sqrtPriceX48 is zero — that
// triggers the linear-fallback path in callers.
func concentrationQ48(
	dst, sqrtPriceX48 *uint256.Int,
	amountIn, reserveX, reserveY *uint256.Int,
	kQ12 uint32,
	xToY bool,
) *uint256.Int {
	if amountIn.IsZero() || kQ12 == 0 || sqrtPriceX48.IsZero() {
		dst.Clear()
		return dst
	}

	var xWealthInY, totalWealthInY, scratch uint256.Int
	big256.MulDivDown(&scratch, reserveX, sqrtPriceX48, q48)
	big256.MulDivDown(&xWealthInY, &scratch, sqrtPriceX48, q48)
	totalWealthInY.Add(&xWealthInY, reserveY)
	if totalWealthInY.IsZero() {
		dst.Clear()
		return dst
	}

	var amountInWealth uint256.Int
	if xToY {
		big256.MulDivDown(&scratch, amountIn, sqrtPriceX48, q48)
		big256.MulDivDown(&amountInWealth, &scratch, sqrtPriceX48, q48)
	} else {
		amountInWealth.Set(amountIn)
	}

	var rQ48 uint256.Int
	if !amountInWealth.Lt(&totalWealthInY) {
		rQ48.Set(q48)
	} else {
		big256.MulDivDown(&rQ48, &amountInWealth, q48, &totalWealthInY)
	}

	var rSquaredQ48 uint256.Int
	big256.MulDivDown(&rSquaredQ48, &rQ48, &rQ48, q48)

	var kU uint256.Int
	kU.SetUint64(uint64(kQ12))
	big256.MulDivDown(dst, &kU, &rSquaredQ48, q12)
	if !dst.Lt(q48) {
		dst.Set(q48)
	}
	return dst
}

func lowerBound(dst, sqrtPriceX48 *uint256.Int, cQ48 uint64) *uint256.Int {
	var oneMinusC, sqrtOneMinusC uint256.Int
	oneMinusC.Sub(q48, oneMinusC.SetUint64(cQ48))
	sqrtOneMinusC.Sqrt(&oneMinusC)
	return big256.MulDivDown(dst, sqrtPriceX48, &sqrtOneMinusC, q24)
}

func upperBound(dst, sqrtPriceX48 *uint256.Int, cQ48 uint64) *uint256.Int {
	var oneMinusC, sqrtOneMinusC uint256.Int
	oneMinusC.Sub(q48, oneMinusC.SetUint64(cQ48))
	sqrtOneMinusC.Sqrt(&oneMinusC)
	return big256.MulDivDown(dst, sqrtPriceX48, q24, &sqrtOneMinusC)
}

// liquidityY = reserveY * Q48 / (sqrtPriceX48 - pBid).
func liquidityY(dst, sqrtPriceX48, pBid, reserveY *uint256.Int) *uint256.Int {
	var denom uint256.Int
	denom.Sub(sqrtPriceX48, pBid)
	return big256.MulDivDown(dst, reserveY, q48, &denom)
}

// liquidityX = reserveX * (sqrtPriceX48 * pAsk) / (Q48 * (pAsk - sqrtPriceX48)).
// Avoids the intermediate /Q96 truncation that bites at small sqrt-prices.
func liquidityX(dst, sqrtPriceX48, pAsk, reserveX *uint256.Int) *uint256.Int {
	var numerator, denominator, diff uint256.Int
	numerator.Mul(sqrtPriceX48, pAsk)
	diff.Sub(pAsk, sqrtPriceX48)
	denominator.Mul(q48, &diff)
	return big256.MulDivDown(dst, reserveX, &numerator, &denominator)
}

// getNextSqrtPriceFromAmountXRoundingUp ports the addX=true branch of Uniswap
// V3 SqrtPriceMath, used by quoteXToY.
func getNextSqrtPriceFromAmountXRoundingUp(dst, sqrtPX48, liquidity, amountX *uint256.Int) *uint256.Int {
	if amountX.IsZero() {
		return dst.Set(sqrtPX48)
	}

	var num1, prod, tmp uint256.Int
	num1.Lsh(liquidity, fixedPoint48Resolution)
	prod.Mul(amountX, sqrtPX48)

	if quotient := tmp.Div(&prod, amountX); quotient.Eq(sqrtPX48) {
		var deno uint256.Int
		deno.Add(&num1, &prod)
		if !deno.Lt(&num1) {
			return big256.MulDivUp(dst, &num1, sqrtPX48, &deno)
		}
	}

	var divResult, deno uint256.Int
	divResult.Div(&num1, sqrtPX48)
	deno.Add(&divResult, amountX)
	return ceilDiv(dst, &num1, &deno)
}

// getNextSqrtPriceFromAmountYRoundingDown ports the addY=true branch of
// Uniswap V3 SqrtPriceMath, used by quoteYToX.
func getNextSqrtPriceFromAmountYRoundingDown(dst, sqrtPX48, liquidity, amountY *uint256.Int) *uint256.Int {
	var quotient uint256.Int
	if amountY.Lt(u2Pow80) {
		var shifted uint256.Int
		shifted.Lsh(amountY, fixedPoint48Resolution)
		quotient.Div(&shifted, liquidity)
	} else {
		big256.MulDivDown(&quotient, amountY, q48, liquidity)
	}
	return dst.Add(sqrtPX48, &quotient)
}

func getAmountXDelta(dst, sqrtRatioA, sqrtRatioB, liquidity *uint256.Int, roundUp bool) *uint256.Int {
	sa, sb := sqrtRatioA, sqrtRatioB
	if sa.Gt(sb) {
		sa, sb = sb, sa
	}

	var num1, num2 uint256.Int
	num1.Lsh(liquidity, fixedPoint48Resolution)
	num2.Sub(sb, sa)

	if roundUp {
		big256.MulDivUp(dst, &num1, &num2, sb)
		return ceilDiv(dst, dst, sa)
	}
	big256.MulDivDown(dst, &num1, &num2, sb)
	return dst.Div(dst, sa)
}

func getAmountYDelta(dst, sqrtRatioA, sqrtRatioB, liquidity *uint256.Int, roundUp bool) *uint256.Int {
	sa, sb := sqrtRatioA, sqrtRatioB
	if sa.Gt(sb) {
		sa, sb = sb, sa
	}

	var diff uint256.Int
	diff.Sub(sb, sa)

	if roundUp {
		return big256.MulDivUp(dst, liquidity, &diff, q48)
	}
	return big256.MulDivDown(dst, liquidity, &diff, q48)
}

// quoteXToY is a bit-exact port of Solidity SwapLib._quoteXToY (single-price
// Q32.48 design with asymmetric directional fees and linear fallback).
func quoteXToY(params *PoolParams, dx *uint256.Int) *QuoteResult {
	out := &QuoteResult{
		AmountOut:     new(uint256.Int),
		SqrtPriceNext: new(uint256.Int),
		Fee:           new(uint256.Int),
	}
	quoteXToYInto(out, params, dx)
	return out
}

func quoteYToX(params *PoolParams, dy *uint256.Int) *QuoteResult {
	out := &QuoteResult{
		AmountOut:     new(uint256.Int),
		SqrtPriceNext: new(uint256.Int),
		Fee:           new(uint256.Int),
	}
	quoteYToXInto(out, params, dy)
	return out
}

func quoteXToYInto(out *QuoteResult, params *PoolParams, dx *uint256.Int) *QuoteResult {
	var (
		cQ48      uint256.Int
		pBid      uint256.Int
		liquidity uint256.Int
		maxNetDx  uint256.Int
		pNext     uint256.Int
		dy        uint256.Int
		feeQ24    uint256.Int
	)

	concentrationQ48(&cQ48, params.SqrtPriceX48, dx,
		params.ReserveX, params.ReserveY, params.ConcentrationK, true)
	if cQ48.IsZero() {
		linearXToY(out, params, dx)
		return out
	}
	if !cQ48.Lt(q48) {
		return writeRejected(out, params)
	}

	lowerBound(&pBid, params.SqrtPriceX48, cQ48.Uint64())
	if !params.SqrtPriceX48.Gt(&pBid) {
		return writeRejected(out, params)
	}
	liquidityY(&liquidity, params.SqrtPriceX48, &pBid, params.ReserveY)

	getAmountXDelta(&maxNetDx, &pBid, params.SqrtPriceX48, &liquidity, false)
	if dx.Gt(&maxNetDx) {
		return writeRejected(out, params)
	}

	getNextSqrtPriceFromAmountXRoundingUp(&pNext, params.SqrtPriceX48, &liquidity, dx)
	getAmountYDelta(&dy, params.SqrtPriceX48, &pNext, &liquidity, false)

	feeQ24.SetUint64(uint64(params.FeeBidX24))
	big256.MulDivDown(out.Fee, &dy, &feeQ24, q24)
	out.AmountOut.Sub(&dy, out.Fee)
	out.SqrtPriceNext.Set(&pNext)
	return out
}

func quoteYToXInto(out *QuoteResult, params *PoolParams, dy *uint256.Int) *QuoteResult {
	var (
		cQ48      uint256.Int
		pAsk      uint256.Int
		liquidity uint256.Int
		maxNetDy  uint256.Int
		pNext     uint256.Int
		dxOut     uint256.Int
		feeQ24    uint256.Int
	)

	concentrationQ48(&cQ48, params.SqrtPriceX48, dy,
		params.ReserveX, params.ReserveY, params.ConcentrationK, false)
	if cQ48.IsZero() {
		linearYToX(out, params, dy)
		return out
	}
	if !cQ48.Lt(q48) {
		return writeRejected(out, params)
	}

	upperBound(&pAsk, params.SqrtPriceX48, cQ48.Uint64())
	if !params.SqrtPriceX48.Lt(&pAsk) {
		return writeRejected(out, params)
	}
	liquidityX(&liquidity, params.SqrtPriceX48, &pAsk, params.ReserveX)

	getAmountYDelta(&maxNetDy, params.SqrtPriceX48, &pAsk, &liquidity, false)
	if dy.Gt(&maxNetDy) {
		return writeRejected(out, params)
	}

	getNextSqrtPriceFromAmountYRoundingDown(&pNext, params.SqrtPriceX48, &liquidity, dy)
	getAmountXDelta(&dxOut, params.SqrtPriceX48, &pNext, &liquidity, false)

	feeQ24.SetUint64(uint64(params.FeeAskX24))
	big256.MulDivDown(out.Fee, &dxOut, &feeQ24, q24)
	out.AmountOut.Sub(&dxOut, out.Fee)
	out.SqrtPriceNext.Set(&pNext)
	return out
}

// linearXToY implements the cQ48 == 0 fallback for X → Y:
// dy = mulDiv(mulDiv(dx, sqrtPriceX48, Q48), sqrtPriceX48, Q48),
// fee on dy, pNext = sqrtPriceX48.
func linearXToY(out *QuoteResult, params *PoolParams, dx *uint256.Int) {
	var dyGross, scratch, feeQ24 uint256.Int
	big256.MulDivDown(&scratch, dx, params.SqrtPriceX48, q48)
	big256.MulDivDown(&dyGross, &scratch, params.SqrtPriceX48, q48)
	if dyGross.IsZero() || dyGross.Gt(params.ReserveY) {
		writeRejected(out, params)
		return
	}
	feeQ24.SetUint64(uint64(params.FeeBidX24))
	big256.MulDivDown(out.Fee, &dyGross, &feeQ24, q24)
	out.AmountOut.Sub(&dyGross, out.Fee)
	out.SqrtPriceNext.Set(params.SqrtPriceX48)
}

func linearYToX(out *QuoteResult, params *PoolParams, dy *uint256.Int) {
	if params.SqrtPriceX48.IsZero() {
		writeRejected(out, params)
		return
	}
	var dxGross, scratch, feeQ24 uint256.Int
	big256.MulDivDown(&scratch, dy, q48, params.SqrtPriceX48)
	big256.MulDivDown(&dxGross, &scratch, q48, params.SqrtPriceX48)
	if dxGross.IsZero() || dxGross.Gt(params.ReserveX) {
		writeRejected(out, params)
		return
	}
	feeQ24.SetUint64(uint64(params.FeeAskX24))
	big256.MulDivDown(out.Fee, &dxGross, &feeQ24, q24)
	out.AmountOut.Sub(&dxGross, out.Fee)
	out.SqrtPriceNext.Set(params.SqrtPriceX48)
}

func writeRejected(out *QuoteResult, params *PoolParams) *QuoteResult {
	out.AmountOut.Clear()
	out.Fee.Clear()
	out.SqrtPriceNext.Set(params.SqrtPriceX48)
	return out
}

// SqrtPriceX48ToX96 lifts a Q32.48 sqrt-price (uint80) into a Q64.96 sqrt-
// price (uint160) by shifting left 48 bits. Lossless. Retained for legacy
// callers carrying pre-Q48 serialised state.
func SqrtPriceX48ToX96(pX48 *uint256.Int) *uint256.Int {
	if pX48 == nil {
		return nil
	}
	out := new(uint256.Int).Set(pX48)
	return out.Lsh(out, 48)
}

// SqrtPriceX96ToX48 lowers a Q64.96 sqrt-price into a Q32.48 sqrt-price by
// right-shifting 48 bits, truncating the bottom 48 bits of precision.
func SqrtPriceX96ToX48(pX96 *uint256.Int) *uint256.Int {
	if pX96 == nil {
		return nil
	}
	out := new(uint256.Int).Set(pX96)
	return out.Rsh(out, 48)
}

// PriceToSqrtPriceX48 converts a plain decimal price into a Q32.48 sqrt-price
// (uint80) as *uint256.Int. Lossy beyond float64's 53-bit significand. Panics
// on NaN/Inf/negative; saturates at 2^80-1 on overflow.
func PriceToSqrtPriceX48(price float64) *uint256.Int {
	if math.IsNaN(price) || math.IsInf(price, 0) || price < 0 {
		panic("price must be finite and non-negative")
	}
	scaled := math.Sqrt(price) * math.Pow(2, 48)
	u80Max := new(uint256.Int).Sub(new(uint256.Int).Lsh(big256.U1, 80), big256.U1)
	if math.IsInf(scaled, 0) {
		return new(uint256.Int)
	}
	if scaled >= math.Ldexp(1, 80) {
		return u80Max
	}
	return new(uint256.Int).SetUint64(uint64(scaled))
}

// SqrtPriceX48ToPrice converts a Q32.48 sqrt-price back to a plain decimal
// price. Pass nil through as 0.
func SqrtPriceX48ToPrice(pX48 *uint256.Int) float64 {
	if pX48 == nil {
		return 0
	}
	sqrtP := u256ToF64Lossy(pX48) / math.Pow(2, 48)
	return sqrtP * sqrtP
}

// PlainToQ12ConcentrationK lifts a plain effective K into the Q20.12
// representation expected by PoolParams.ConcentrationK. Saturates at
// math.MaxUint32.
func PlainToQ12ConcentrationK(k uint32) uint32 {
	const limit = uint32(1) << 20
	if k >= limit {
		return ^uint32(0)
	}
	return k << 12
}

// Q12ToPlainConcentrationK reverses PlainToQ12ConcentrationK (truncates).
func Q12ToPlainConcentrationK(kQ12 uint32) uint32 {
	return kQ12 >> 12
}

func u256ToF64Lossy(v *uint256.Int) float64 {
	if v.IsZero() {
		return 0
	}
	bitLen := v.BitLen()
	if bitLen <= 64 {
		return float64(v.Uint64())
	}
	shift := uint(bitLen - 53)
	truncated := new(uint256.Int).Rsh(v, shift)
	return float64(truncated.Uint64()) * math.Ldexp(1, int(shift))
}

package lunarbase

import (
	"math"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// Direct port of the on-chain SwapLib Q64.96 / uint160 price math. Bit-for-bit
// identical with the lunarbase off-chain Rust, Node, and Go packages.

const fixedPoint96Resolution = 96

var (
	q12      = new(uint256.Int).Lsh(big256.U1, 12)
	q24      = new(uint256.Int).Lsh(big256.U1, 24)
	q48      = new(uint256.Int).Lsh(big256.U1, 48)
	q96      = new(uint256.Int).Lsh(big256.U1, 96)
	u2Pow160 = new(uint256.Int).Lsh(big256.U1, 160)
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

// concentrationQ48 writes c = mulDiv(concentrationK, r^2, Q12) into dst.
// The wealth normalisation uses Q64.96 sqrt-prices and saturates at Q48.
func concentrationQ48(
	dst, sqrtPriceX96 *uint256.Int,
	amountIn, reserveX, reserveY *uint256.Int,
	kQ12 uint32,
	xToY bool,
) *uint256.Int {
	if amountIn.IsZero() || kQ12 == 0 || sqrtPriceX96.IsZero() {
		dst.Clear()
		return dst
	}

	var xWealthInY, totalWealthInY, scratch uint256.Int
	big256.MulDivDown(&scratch, reserveX, sqrtPriceX96, q96)
	big256.MulDivDown(&xWealthInY, &scratch, sqrtPriceX96, q96)
	totalWealthInY.Add(&xWealthInY, reserveY)
	if totalWealthInY.IsZero() {
		dst.Clear()
		return dst
	}

	var amountInWealth uint256.Int
	if xToY {
		big256.MulDivDown(&scratch, amountIn, sqrtPriceX96, q96)
		big256.MulDivDown(&amountInWealth, &scratch, sqrtPriceX96, q96)
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

func lowerBound(dst, sqrtPriceX96 *uint256.Int, cQ48 uint64) *uint256.Int {
	var oneMinusC, sqrtOneMinusC uint256.Int
	oneMinusC.Sub(q48, oneMinusC.SetUint64(cQ48))
	sqrtOneMinusC.Sqrt(&oneMinusC)
	return big256.MulDivDown(dst, sqrtPriceX96, &sqrtOneMinusC, q24)
}

func upperBound(dst, sqrtPriceX96 *uint256.Int, cQ48 uint64) *uint256.Int {
	var oneMinusC, sqrtOneMinusC uint256.Int
	oneMinusC.Sub(q48, oneMinusC.SetUint64(cQ48))
	sqrtOneMinusC.Sqrt(&oneMinusC)
	return big256.MulDivDown(dst, sqrtPriceX96, q24, &sqrtOneMinusC)
}

func liquidityY(dst, sqrtPriceX96, pBid, reserveY *uint256.Int) *uint256.Int {
	var denom uint256.Int
	denom.Sub(sqrtPriceX96, pBid)
	return big256.MulDivDown(dst, reserveY, q96, &denom)
}

func liquidityX(dst, sqrtPriceX96, pAsk, reserveX *uint256.Int) *uint256.Int {
	var priceProductX96, diff uint256.Int
	big256.MulDivDown(&priceProductX96, sqrtPriceX96, pAsk, q96)
	diff.Sub(pAsk, sqrtPriceX96)
	return big256.MulDivDown(dst, reserveX, &priceProductX96, &diff)
}

func getNextSqrtPriceFromAmountXRoundingUp(dst, sqrtPX96, liquidity, amountX *uint256.Int) *uint256.Int {
	if amountX.IsZero() {
		return dst.Set(sqrtPX96)
	}

	var num1, prod, tmp uint256.Int
	num1.Lsh(liquidity, fixedPoint96Resolution)
	prod.Mul(amountX, sqrtPX96)

	if quotient := tmp.Div(&prod, amountX); quotient.Eq(sqrtPX96) {
		var deno uint256.Int
		deno.Add(&num1, &prod)
		if !deno.Lt(&num1) {
			return big256.MulDivUp(dst, &num1, sqrtPX96, &deno)
		}
	}

	var divResult, deno uint256.Int
	divResult.Div(&num1, sqrtPX96)
	deno.Add(&divResult, amountX)
	return ceilDiv(dst, &num1, &deno)
}

func getNextSqrtPriceFromAmountYRoundingDown(dst, sqrtPX96, liquidity, amountY *uint256.Int) *uint256.Int {
	var quotient uint256.Int
	if amountY.Lt(u2Pow160) {
		var shifted uint256.Int
		shifted.Lsh(amountY, fixedPoint96Resolution)
		quotient.Div(&shifted, liquidity)
	} else {
		big256.MulDivDown(&quotient, amountY, q96, liquidity)
	}
	return dst.Add(sqrtPX96, &quotient)
}

func getAmountXDelta(dst, sqrtRatioA, sqrtRatioB, liquidity *uint256.Int, roundUp bool) *uint256.Int {
	sa, sb := sqrtRatioA, sqrtRatioB
	if sa.Gt(sb) {
		sa, sb = sb, sa
	}

	var num1, num2 uint256.Int
	num1.Lsh(liquidity, fixedPoint96Resolution)
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
		return big256.MulDivUp(dst, liquidity, &diff, q96)
	}
	return big256.MulDivDown(dst, liquidity, &diff, q96)
}

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

	concentrationQ48(&cQ48, params.SqrtPriceX96, dx,
		params.ReserveX, params.ReserveY, params.ConcentrationK, true)
	if cQ48.IsZero() {
		linearXToY(out, params, dx)
		return out
	}
	if !cQ48.Lt(q48) {
		return writeRejected(out, params)
	}

	lowerBound(&pBid, params.SqrtPriceX96, cQ48.Uint64())
	if !params.SqrtPriceX96.Gt(&pBid) {
		return writeRejected(out, params)
	}
	liquidityY(&liquidity, params.SqrtPriceX96, &pBid, params.ReserveY)

	getAmountXDelta(&maxNetDx, &pBid, params.SqrtPriceX96, &liquidity, false)
	if dx.Gt(&maxNetDx) {
		return writeRejected(out, params)
	}

	getNextSqrtPriceFromAmountXRoundingUp(&pNext, params.SqrtPriceX96, &liquidity, dx)
	getAmountYDelta(&dy, params.SqrtPriceX96, &pNext, &liquidity, false)
	if dy.IsZero() {
		linearXToY(out, params, dx)
		return out
	}

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

	concentrationQ48(&cQ48, params.SqrtPriceX96, dy,
		params.ReserveX, params.ReserveY, params.ConcentrationK, false)
	if cQ48.IsZero() {
		linearYToX(out, params, dy)
		return out
	}
	if !cQ48.Lt(q48) {
		return writeRejected(out, params)
	}

	upperBound(&pAsk, params.SqrtPriceX96, cQ48.Uint64())
	if !params.SqrtPriceX96.Lt(&pAsk) {
		return writeRejected(out, params)
	}
	liquidityX(&liquidity, params.SqrtPriceX96, &pAsk, params.ReserveX)

	getAmountYDelta(&maxNetDy, params.SqrtPriceX96, &pAsk, &liquidity, false)
	if dy.Gt(&maxNetDy) {
		return writeRejected(out, params)
	}

	getNextSqrtPriceFromAmountYRoundingDown(&pNext, params.SqrtPriceX96, &liquidity, dy)
	getAmountXDelta(&dxOut, params.SqrtPriceX96, &pNext, &liquidity, false)
	if dxOut.IsZero() {
		linearYToX(out, params, dy)
		return out
	}

	feeQ24.SetUint64(uint64(params.FeeAskX24))
	big256.MulDivDown(out.Fee, &dxOut, &feeQ24, q24)
	out.AmountOut.Sub(&dxOut, out.Fee)
	out.SqrtPriceNext.Set(&pNext)
	return out
}

func linearXToY(out *QuoteResult, params *PoolParams, dx *uint256.Int) {
	var dyGross, scratch, feeQ24 uint256.Int
	big256.MulDivDown(&scratch, dx, params.SqrtPriceX96, q96)
	big256.MulDivDown(&dyGross, &scratch, params.SqrtPriceX96, q96)
	if dyGross.IsZero() || dyGross.Gt(params.ReserveY) {
		writeRejected(out, params)
		return
	}
	feeQ24.SetUint64(uint64(params.FeeBidX24))
	big256.MulDivDown(out.Fee, &dyGross, &feeQ24, q24)
	out.AmountOut.Sub(&dyGross, out.Fee)
	out.SqrtPriceNext.Set(params.SqrtPriceX96)
}

func linearYToX(out *QuoteResult, params *PoolParams, dy *uint256.Int) {
	if params.SqrtPriceX96.IsZero() {
		writeRejected(out, params)
		return
	}
	var dxGross, scratch, feeQ24 uint256.Int
	big256.MulDivDown(&scratch, dy, q96, params.SqrtPriceX96)
	big256.MulDivDown(&dxGross, &scratch, q96, params.SqrtPriceX96)
	if dxGross.IsZero() || dxGross.Gt(params.ReserveX) {
		writeRejected(out, params)
		return
	}
	feeQ24.SetUint64(uint64(params.FeeAskX24))
	big256.MulDivDown(out.Fee, &dxGross, &feeQ24, q24)
	out.AmountOut.Sub(&dxGross, out.Fee)
	out.SqrtPriceNext.Set(params.SqrtPriceX96)
}

func writeRejected(out *QuoteResult, params *PoolParams) *QuoteResult {
	out.AmountOut.Clear()
	out.Fee.Clear()
	out.SqrtPriceNext.Set(params.SqrtPriceX96)
	return out
}

// PriceToSqrtPriceX96 converts a plain decimal price into a Q64.96 sqrt-price.
func PriceToSqrtPriceX96(price float64) *uint256.Int {
	if math.IsNaN(price) || math.IsInf(price, 0) || price < 0 {
		panic("price must be finite and non-negative")
	}
	scaled := math.Sqrt(price) * math.Pow(2, 96)
	return f64FloorToU256(scaled)
}

// SqrtPriceX96ToPrice converts a Q64.96 sqrt-price back to a plain price.
func SqrtPriceX96ToPrice(pX96 *uint256.Int) float64 {
	if pX96 == nil {
		return 0
	}
	sqrtP := u256ToF64Lossy(pX96) / math.Pow(2, 96)
	return sqrtP * sqrtP
}

func PlainToQ12ConcentrationK(k uint32) uint32 {
	const limit = uint32(1) << 20
	if k >= limit {
		return ^uint32(0)
	}
	return k << 12
}

func Q12ToPlainConcentrationK(kQ12 uint32) uint32 {
	return kQ12 >> 12
}

func f64FloorToU256(x float64) *uint256.Int {
	if math.IsNaN(x) || math.IsInf(x, 0) || x < 1 {
		return new(uint256.Int)
	}
	bits := math.Float64bits(x)
	exp := int((bits>>52)&0x7ff) - 1023
	mantissa := (bits & ((1 << 52) - 1)) | (1 << 52)
	out := new(uint256.Int).SetUint64(mantissa)
	if exp >= 52 {
		shift := uint(exp - 52)
		if shift >= 256-53 {
			max := new(uint256.Int)
			max.Not(max)
			return max
		}
		return out.Lsh(out, shift)
	}
	return out.Rsh(out, uint(52-exp))
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

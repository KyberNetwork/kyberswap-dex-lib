package lunarbase

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	q48  = new(uint256.Int).Lsh(big256.U1, 48)
	q24  = new(uint256.Int).Lsh(big256.U1, 24)
	q120 = new(uint256.Int).Lsh(big256.U1, 120)
)

func isqrt(x *uint256.Int) *uint256.Int {
	return new(uint256.Int).Sqrt(x)
}

func concentrationQ48(baseFeeQ48 uint64, amountIn *uint256.Int, reserveIn *uint256.Int, k uint32) *uint256.Int {
	c := uint256.NewInt(baseFeeQ48)
	if c.IsZero() || amountIn.IsZero() || reserveIn.IsZero() || k == 0 {
		return c
	}

	var rQ48 uint256.Int
	if !amountIn.Lt(reserveIn) {
		rQ48.Set(q48)
	} else {
		big256.MulDivDown(&rQ48, amountIn, q48, reserveIn)
	}

	var rSquaredQ48 uint256.Int
	big256.MulDivDown(&rSquaredQ48, &rQ48, &rQ48, q48)

	kU := uint256.NewInt(uint64(k))
	var kTimesR2 uint256.Int
	kTimesR2.Mul(kU, &rSquaredQ48)
	var multiplierQ48 uint256.Int
	multiplierQ48.Add(q48, &kTimesR2)

	var result uint256.Int
	big256.MulDivDown(&result, c, &multiplierQ48, q48)

	if !result.Lt(q48) {
		return result.Set(q48)
	}
	return &result
}

func computeLiquidity(sqrtPriceX96 *uint256.Int, cQ48 uint64, reserveX, reserveY *uint256.Int) *uint256.Int {
	var tmp uint256.Int
	oneMinusC := tmp.Sub(q48, tmp.SetUint64(cQ48))
	qX24 := isqrt(oneMinusC)
	denomX24 := tmp.Sub(q24, qX24)
	if denomX24.IsZero() {
		return big256.U0
	}

	denomLy := qX24.Mul(sqrtPriceX96, denomX24)
	ly := big256.MulDivDown(denomLy, reserveY, q120, denomLy)

	denomLx := tmp.Lsh(denomX24, 72)
	lx := big256.MulDivDown(denomLx, reserveX, sqrtPriceX96, denomLx)

	return big256.Min(lx, ly)
}

func getNextSqrtPriceFromAmountXRoundingUp(sqrtPX96, liquidity, amountX *uint256.Int) *uint256.Int {
	if amountX.IsZero() {
		return sqrtPX96
	}

	var num, prod, tmp uint256.Int
	num.Lsh(liquidity, 96)
	prod.Mul(amountX, sqrtPX96)

	if quotient := tmp.Div(&prod, amountX); quotient.Eq(sqrtPX96) {
		if deno := tmp.Add(&num, &prod); !deno.Lt(&num) {
			return big256.MulDivUp(deno, &num, sqrtPX96, deno)
		}
	}

	divResult := num.Div(&num, sqrtPX96)
	deno := divResult.Add(divResult, amountX)
	return ceilDiv(&num, deno)
}

func getNextSqrtPriceFromAmountYRoundingDown(sqrtPX96, liquidity, amountY *uint256.Int) *uint256.Int {
	var quotient uint256.Int
	if amountY.Lt(big256.U2Pow160) {
		// (amountY << 96) / liquidity
		shifted := quotient.Lsh(amountY, 96)
		quotient.Div(shifted, liquidity)
	} else {
		big256.MulDivDown(&quotient, amountY, q96, liquidity)
	}

	return quotient.Add(sqrtPX96, &quotient)
}

func getAmountXDelta(sqrtRatioA, sqrtRatioB, liquidity *uint256.Int, roundUp bool) *uint256.Int {
	sa, sb := sqrtRatioA, sqrtRatioB
	if sa.Gt(sb) {
		sa, sb = sb, sa
	}

	var num1, num2 uint256.Int
	num1.Lsh(liquidity, 96)
	num2.Sub(sb, sa)

	if roundUp {
		md := big256.MulDivUp(&num1, &num1, &num2, sb)
		return ceilDiv(md, sa)
	}
	md := big256.MulDivDown(&num1, &num1, &num2, sb)
	return md.Div(md, sa)
}

func getAmountYDelta(sqrtRatioA, sqrtRatioB, liquidity *uint256.Int, roundUp bool) *uint256.Int {
	sa, sb := sqrtRatioA, sqrtRatioB
	if sa.Gt(sb) {
		sa, sb = sb, sa
	}

	var diff uint256.Int
	diff.Sub(sb, sa)

	if roundUp {
		return big256.MulDivUp(&diff, liquidity, &diff, q96)
	}
	return big256.MulDivDown(&diff, liquidity, &diff, q96)
}

func ceilDiv(a, b *uint256.Int) *uint256.Int {
	var q, rem uint256.Int
	if q.DivMod(a, b, &rem); !rem.IsZero() {
		q.AddUint64(&q, 1)
	}
	return &q
}

func quoteXToY(params *PoolParams, dx *uint256.Int) *QuoteResult {
	zero := &QuoteResult{
		AmountOut:     big256.U0,
		SqrtPriceNext: params.SqrtPriceX96.Clone(),
		Fee:           big256.U0,
	}

	cQ48 := concentrationQ48(params.FeeQ48, dx, params.ReserveX, params.ConcentrationK)
	if !cQ48.Lt(q48) {
		return zero
	}

	cU64 := cQ48.Uint64()
	liquidity := computeLiquidity(params.SqrtPriceX96, cU64, params.ReserveX, params.ReserveY)

	oneMinusCQ48 := new(uint256.Int).Sub(q48, cQ48)
	sqrtOneMinusC := isqrt(oneMinusCQ48)
	pBid := big256.MulDivDown(oneMinusCQ48, params.SqrtPriceX96, sqrtOneMinusC, q24)

	maxNetDx := getAmountXDelta(pBid, params.SqrtPriceX96, liquidity, false)
	if dx.Gt(maxNetDx) {
		return zero
	}

	pNext := getNextSqrtPriceFromAmountXRoundingUp(params.SqrtPriceX96, liquidity, dx)
	dy := getAmountYDelta(params.SqrtPriceX96, pNext, liquidity, false)

	fee := big256.MulDivDown(sqrtOneMinusC, dy, sqrtOneMinusC.SetUint64(params.FeeQ48), q48)
	dyAfterFee := dy.Sub(dy, fee)

	return &QuoteResult{
		AmountOut:     dyAfterFee,
		SqrtPriceNext: pNext,
		Fee:           fee,
	}
}

func quoteYToX(params *PoolParams, dy *uint256.Int) *QuoteResult {
	zero := &QuoteResult{
		AmountOut:     big256.U0,
		SqrtPriceNext: params.SqrtPriceX96.Clone(),
		Fee:           big256.U0,
	}

	cQ48 := concentrationQ48(params.FeeQ48, dy, params.ReserveY, params.ConcentrationK)
	if !cQ48.Lt(q48) {
		return zero
	}

	cU64 := cQ48.Uint64()
	liquidity := computeLiquidity(params.SqrtPriceX96, cU64, params.ReserveX, params.ReserveY)

	oneMinusCQ48 := new(uint256.Int).Sub(q48, cQ48)
	sqrtOneMinusC := isqrt(oneMinusCQ48)
	pAsk := big256.MulDivDown(oneMinusCQ48, params.SqrtPriceX96, q24, sqrtOneMinusC)

	maxNetDy := getAmountYDelta(params.SqrtPriceX96, pAsk, liquidity, false)
	if dy.Gt(maxNetDy) {
		return zero
	}

	pNext := getNextSqrtPriceFromAmountYRoundingDown(params.SqrtPriceX96, liquidity, dy)
	dxOut := getAmountXDelta(params.SqrtPriceX96, pNext, liquidity, false)

	fee := big256.MulDivDown(sqrtOneMinusC, dxOut, sqrtOneMinusC.SetUint64(params.FeeQ48), q48)
	dxAfterFee := dxOut.Sub(dxOut, fee)

	return &QuoteResult{
		AmountOut:     dxAfterFee,
		SqrtPriceNext: pNext,
		Fee:           fee,
	}
}

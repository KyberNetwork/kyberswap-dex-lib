package lunarbase

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const fixedPoint48Resolution = 48

var (
	q48     = new(uint256.Int).Lsh(big256.U1, 48)
	q24     = new(uint256.Int).Lsh(big256.U1, 24)
	u2Pow80 = new(uint256.Int).Lsh(big256.U1, 80)
)

func isqrt(x *uint256.Int) *uint256.Int {
	return new(uint256.Int).Sqrt(x)
}

// concentrationQ48 mirrors SwapLib.concentrationQ48 (wealth-normalized r).
func concentrationQ48(
	pX48 *uint256.Int,
	baseFeeQ48 uint64,
	amountIn *uint256.Int,
	reserveX, reserveY *uint256.Int,
	k uint32,
	xToY bool,
) *uint256.Int {
	c := uint256.NewInt(baseFeeQ48)
	if c.IsZero() || amountIn.IsZero() || k == 0 || pX48.IsZero() {
		return c
	}

	var priceQ96, q96 uint256.Int
	priceQ96.Mul(pX48, pX48)
	q96.Mul(q48, q48)

	var xWealthInY uint256.Int
	big256.MulDivDown(&xWealthInY, reserveX, &priceQ96, &q96)

	var totalWealthInY uint256.Int
	totalWealthInY.Add(&xWealthInY, reserveY)
	if totalWealthInY.IsZero() {
		return c
	}

	var amountInWealth uint256.Int
	if xToY {
		big256.MulDivDown(&amountInWealth, amountIn, &priceQ96, &q96)
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

func lowerBound(pX48 *uint256.Int, cQ48 uint64) *uint256.Int {
	var oneMinusC uint256.Int
	oneMinusC.Sub(q48, oneMinusC.SetUint64(cQ48))

	sqrtOneMinusC := isqrt(&oneMinusC)
	var result uint256.Int
	return big256.MulDivDown(&result, pX48, sqrtOneMinusC, q24)
}

func upperBound(pX48 *uint256.Int, cQ48 uint64) *uint256.Int {
	var oneMinusC uint256.Int
	oneMinusC.Sub(q48, oneMinusC.SetUint64(cQ48))

	sqrtOneMinusC := isqrt(&oneMinusC)
	var result uint256.Int
	return big256.MulDivDown(&result, pX48, q24, sqrtOneMinusC)
}

func liquidityY(pX48, pBid, reserveY *uint256.Int) *uint256.Int {
	var denom uint256.Int
	denom.Sub(pX48, pBid)

	var result uint256.Int
	return big256.MulDivDown(&result, reserveY, q48, &denom)
}

// liquidityX mirrors SwapLib.Lx: mulDiv(reserveX, pX48*pAsk, Q48*(pAsk-pX48)).
func liquidityX(pX48, pAsk, reserveX *uint256.Int) *uint256.Int {
	var num uint256.Int
	num.Mul(pX48, pAsk)

	var denom uint256.Int
	denom.Sub(pAsk, pX48)
	denom.Mul(q48, &denom)

	var result uint256.Int
	return big256.MulDivDown(&result, reserveX, &num, &denom)
}

// getNextSqrtPriceFromAmountXRoundingUp ports the addX=true branch of SqrtPriceMath.
func getNextSqrtPriceFromAmountXRoundingUp(sqrtPX48, liquidity, amountX *uint256.Int) *uint256.Int {
	if amountX.IsZero() {
		return new(uint256.Int).Set(sqrtPX48)
	}

	var num1, prod, tmp uint256.Int
	num1.Lsh(liquidity, fixedPoint48Resolution)
	prod.Mul(amountX, sqrtPX48)

	if quotient := tmp.Div(&prod, amountX); quotient.Eq(sqrtPX48) {
		var deno uint256.Int
		deno.Add(&num1, &prod)
		if !deno.Lt(&num1) {
			return big256.MulDivUp(new(uint256.Int), &num1, sqrtPX48, &deno)
		}
	}

	var divResult uint256.Int
	divResult.Div(&num1, sqrtPX48)
	var deno uint256.Int
	deno.Add(&divResult, amountX)
	return ceilDiv(&num1, &deno)
}

// getNextSqrtPriceFromAmountYRoundingDown ports the addY=true branch of SqrtPriceMath.
func getNextSqrtPriceFromAmountYRoundingDown(sqrtPX48, liquidity, amountY *uint256.Int) *uint256.Int {
	var quotient uint256.Int
	if amountY.Lt(u2Pow80) {
		shifted := new(uint256.Int).Lsh(amountY, fixedPoint48Resolution)
		quotient.Div(shifted, liquidity)
	} else {
		big256.MulDivDown(&quotient, amountY, q48, liquidity)
	}

	return new(uint256.Int).Add(sqrtPX48, &quotient)
}

func getAmountXDelta(sqrtRatioA, sqrtRatioB, liquidity *uint256.Int, roundUp bool) *uint256.Int {
	sa, sb := sqrtRatioA, sqrtRatioB
	if sa.Gt(sb) {
		sa, sb = sb, sa
	}

	var num1, num2 uint256.Int
	num1.Lsh(liquidity, fixedPoint48Resolution)
	num2.Sub(sb, sa)

	if roundUp {
		md := big256.MulDivUp(new(uint256.Int), &num1, &num2, sb)
		return ceilDiv(md, sa)
	}
	md := big256.MulDivDown(new(uint256.Int), &num1, &num2, sb)
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
		return big256.MulDivUp(new(uint256.Int), liquidity, &diff, q48)
	}
	return big256.MulDivDown(new(uint256.Int), liquidity, &diff, q48)
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
		SqrtPriceNext: params.SqrtPriceX48.Clone(),
		Fee:           big256.U0,
	}

	cQ48 := concentrationQ48(params.SqrtPriceX48, params.FeeQ48, dx, params.ReserveX, params.ReserveY, params.ConcentrationK, true)
	if !cQ48.Lt(q48) {
		return zero
	}

	cU64 := cQ48.Uint64()
	pBid := lowerBound(params.AnchorSqrtPriceX48, cU64)
	if !params.SqrtPriceX48.Gt(pBid) {
		return zero
	}
	liquidity := liquidityY(params.SqrtPriceX48, pBid, params.ReserveY)

	maxNetDx := getAmountXDelta(pBid, params.SqrtPriceX48, liquidity, false)
	if dx.Gt(maxNetDx) {
		return zero
	}

	pNext := getNextSqrtPriceFromAmountXRoundingUp(params.SqrtPriceX48, liquidity, dx)
	dy := getAmountYDelta(params.SqrtPriceX48, pNext, liquidity, false)

	fee := big256.MulDivDown(new(uint256.Int), dy, uint256.NewInt(params.FeeQ48), q48)
	dyAfterFee := new(uint256.Int).Sub(dy, fee)

	return &QuoteResult{
		AmountOut:     dyAfterFee,
		SqrtPriceNext: pNext,
		Fee:           fee,
	}
}

func quoteYToX(params *PoolParams, dy *uint256.Int) *QuoteResult {
	zero := &QuoteResult{
		AmountOut:     big256.U0,
		SqrtPriceNext: params.SqrtPriceX48.Clone(),
		Fee:           big256.U0,
	}

	cQ48 := concentrationQ48(params.SqrtPriceX48, params.FeeQ48, dy, params.ReserveX, params.ReserveY, params.ConcentrationK, false)
	if !cQ48.Lt(q48) {
		return zero
	}

	cU64 := cQ48.Uint64()
	pAsk := upperBound(params.AnchorSqrtPriceX48, cU64)
	if !params.SqrtPriceX48.Lt(pAsk) {
		return zero
	}
	liquidity := liquidityX(params.SqrtPriceX48, pAsk, params.ReserveX)

	maxNetDy := getAmountYDelta(params.SqrtPriceX48, pAsk, liquidity, false)
	if dy.Gt(maxNetDy) {
		return zero
	}

	pNext := getNextSqrtPriceFromAmountYRoundingDown(params.SqrtPriceX48, liquidity, dy)
	dxOut := getAmountXDelta(params.SqrtPriceX48, pNext, liquidity, false)

	fee := big256.MulDivDown(new(uint256.Int), dxOut, uint256.NewInt(params.FeeQ48), q48)
	dxAfterFee := new(uint256.Int).Sub(dxOut, fee)

	return &QuoteResult{
		AmountOut:     dxAfterFee,
		SqrtPriceNext: pNext,
		Fee:           fee,
	}
}

package lunarbase

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	q48 = new(uint256.Int).Lsh(big256.U1, 48)
	q24 = new(uint256.Int).Lsh(big256.U1, 24)
)

func isqrt(x *uint256.Int) *uint256.Int {
	return new(uint256.Int).Sqrt(x)
}

func concentrationQ48(baseFeeQ48 uint64, amountIn *uint256.Int, reserveIn *uint256.Int, k uint32) *uint256.Int {
	c := uint256.NewInt(baseFeeQ48)
	if c.IsZero() || amountIn.IsZero() || reserveIn.IsZero() || k == 0 {
		return c
	}

	var rQ48, tmp uint256.Int
	if !amountIn.Lt(reserveIn) {
		rQ48.Set(q48)
	} else {
		big256.MulDivDown(&rQ48, amountIn, q48, reserveIn)
	}

	rSquaredQ48 := big256.MulDivDown(&rQ48, &rQ48, &rQ48, q48)
	kTimesR2 := tmp.Mul(tmp.SetUint64(uint64(k)), rSquaredQ48)
	multiplierQ48 := tmp.Add(kTimesR2, q48)

	result := big256.MulDivDown(&tmp, c, multiplierQ48, q48)
	if !result.Lt(q48) {
		return result.Set(q48)
	}
	return result
}

func lowerBound(sqrtPriceX96 *uint256.Int, cQ48 uint64) *uint256.Int {
	var oneMinusC uint256.Int
	oneMinusC.Sub(q48, oneMinusC.SetUint64(cQ48))

	sqrtOneMinusC := isqrt(&oneMinusC)
	return big256.MulDivDown(sqrtOneMinusC, sqrtPriceX96, sqrtOneMinusC, q24)
}

func upperBound(sqrtPriceX96 *uint256.Int, cQ48 uint64) *uint256.Int {
	var oneMinusC uint256.Int
	oneMinusC.Sub(q48, oneMinusC.SetUint64(cQ48))

	sqrtOneMinusC := isqrt(&oneMinusC)
	return big256.MulDivDown(sqrtOneMinusC, sqrtPriceX96, q24, sqrtOneMinusC)
}

func liquidityY(sqrtPriceX96, pBid, reserveY *uint256.Int) *uint256.Int {
	var denom uint256.Int
	denom.Sub(sqrtPriceX96, pBid)

	return big256.MulDivDown(&denom, reserveY, q96, &denom)
}

func liquidityX(sqrtPriceX96, pAsk, reserveX *uint256.Int) *uint256.Int {
	var priceProductQ96 uint256.Int
	big256.MulDivDown(&priceProductQ96, sqrtPriceX96, pAsk, q96)

	var denom uint256.Int
	denom.Sub(pAsk, sqrtPriceX96)

	return big256.MulDivDown(&denom, reserveX, &priceProductQ96, &denom)
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
	return big256.DivUp(&num, deno)
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
		return big256.DivUp(md, sa)
	}
	md := big256.MulDivDown(&num1, &num1, &num2, sb)
	return md.Div(md, sa)
}

func getAmountYDelta(sqrtRatioA, sqrtRatioB, liquidity *uint256.Int, roundUp bool) *uint256.Int {
	var diff uint256.Int
	diff.Abs(diff.Sub(sqrtRatioB, sqrtRatioA))

	return big256.MulDivRounding(&diff, liquidity, &diff, q96, roundUp)
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
	pBid := lowerBound(params.SqrtPriceX96, cU64)
	liquidity := liquidityY(params.SqrtPriceX96, pBid, params.ReserveY)

	maxNetDx := getAmountXDelta(pBid, params.SqrtPriceX96, liquidity, false)
	if dx.Gt(maxNetDx) {
		return zero
	}

	pNext := getNextSqrtPriceFromAmountXRoundingUp(params.SqrtPriceX96, liquidity, dx)
	dy := getAmountYDelta(params.SqrtPriceX96, pNext, liquidity, false)

	var fee uint256.Int
	big256.MulDivDown(&fee, dy, uint256.NewInt(params.FeeQ48), q48)
	dyAfterFee := dy.Sub(dy, &fee)

	return &QuoteResult{
		AmountOut:     dyAfterFee,
		SqrtPriceNext: pNext,
		Fee:           &fee,
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
	pAsk := upperBound(params.SqrtPriceX96, cU64)
	liquidity := liquidityX(params.SqrtPriceX96, pAsk, params.ReserveX)

	maxNetDy := getAmountYDelta(params.SqrtPriceX96, pAsk, liquidity, false)
	if dy.Gt(maxNetDy) {
		return zero
	}

	pNext := getNextSqrtPriceFromAmountYRoundingDown(params.SqrtPriceX96, liquidity, dy)
	dxOut := getAmountXDelta(params.SqrtPriceX96, pNext, liquidity, false)

	var fee uint256.Int
	big256.MulDivDown(&fee, dxOut, uint256.NewInt(params.FeeQ48), q48)
	dxAfterFee := dxOut.Sub(dxOut, &fee)

	return &QuoteResult{
		AmountOut:     dxAfterFee,
		SqrtPriceNext: pNext,
		Fee:           &fee,
	}
}

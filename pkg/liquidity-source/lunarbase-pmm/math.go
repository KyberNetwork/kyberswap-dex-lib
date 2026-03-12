package lunarbase

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	q48  = new(uint256.Int).Lsh(uint256.NewInt(1), 48)
	q24  = new(uint256.Int).Lsh(uint256.NewInt(1), 24)
	q120 = new(uint256.Int).Lsh(uint256.NewInt(1), 120)
)

func isqrt(x *uint256.Int) *uint256.Int {
	if x.IsZero() {
		return new(uint256.Int)
	}

	shift := (uint(x.BitLen()) + 1) / 2
	result := new(uint256.Int).Lsh(uint256.NewInt(1), shift)

	tmp := new(uint256.Int)
	for i := 0; i < 8; i++ {
		tmp.Div(x, result)
		tmp.Add(tmp, result)
		tmp.Rsh(tmp, 1)
		if tmp.Cmp(result) >= 0 {
			return result
		}
		result.Set(tmp)
	}

	return result
}

func concentrationQ48(baseFeeQ48 uint64, amountIn *uint256.Int, reserveIn *uint256.Int, k uint32) *uint256.Int {
	c := uint256.NewInt(baseFeeQ48)
	if c.IsZero() || amountIn.IsZero() || reserveIn.IsZero() || k == 0 {
		return c
	}

	var rQ48 uint256.Int
	if amountIn.Cmp(reserveIn) >= 0 {
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

	if result.Cmp(q48) >= 0 {
		return new(uint256.Int).Set(q48)
	}
	return new(uint256.Int).Set(&result)
}

// L = min(Lx, Ly).
func computeLiquidity(sqrtPriceX96 *uint256.Int, cQ48 uint64, reserveX, reserveY *uint256.Int) *uint256.Int {
	oneMinusC := new(uint256.Int).Sub(q48, uint256.NewInt(cQ48))
	qX24 := isqrt(oneMinusC)
	denomX24 := new(uint256.Int).Sub(q24, qX24)
	if denomX24.IsZero() {
		return new(uint256.Int)
	}

	denomLx := new(uint256.Int).Lsh(denomX24, 72)
	var lx uint256.Int
	big256.MulDivDown(&lx, reserveX, sqrtPriceX96, denomLx)

	denomLy := new(uint256.Int).Mul(sqrtPriceX96, denomX24)
	var ly uint256.Int
	big256.MulDivDown(&ly, reserveY, q120, denomLy)

	if lx.Cmp(&ly) < 0 {
		return new(uint256.Int).Set(&lx)
	}
	return new(uint256.Int).Set(&ly)
}

func getNextSqrtPriceFromAmountXRoundingUp(sqrtPX96, liquidity, amountX *uint256.Int) *uint256.Int {
	if amountX.IsZero() {
		return new(uint256.Int).Set(sqrtPX96)
	}

	numerator1 := new(uint256.Int).Lsh(liquidity, 96)
	product := new(uint256.Int).Mul(amountX, sqrtPX96)

	quotient := new(uint256.Int).Div(product, amountX)
	if quotient.Eq(sqrtPX96) {
		denominator := new(uint256.Int).Add(numerator1, product)
		if denominator.Cmp(numerator1) >= 0 {
			var result uint256.Int
			big256.MulDivUp(&result, numerator1, sqrtPX96, denominator)
			return &result
		}
	}

	divResult := new(uint256.Int).Div(numerator1, sqrtPX96)
	denominator := new(uint256.Int).Add(divResult, amountX)
	return ceilDiv(numerator1, denominator)
}

func getNextSqrtPriceFromAmountYRoundingDown(sqrtPX96, liquidity, amountY *uint256.Int) *uint256.Int {
	var quotient uint256.Int
	u2Pow160 := big256.U2Pow160
	if amountY.Cmp(u2Pow160) < 0 {
		// (amountY << 96) / liquidity
		shifted := new(uint256.Int).Lsh(amountY, 96)
		quotient.Div(shifted, liquidity)
	} else {
		big256.MulDivDown(&quotient, amountY, q96, liquidity)
	}

	result := new(uint256.Int).Add(sqrtPX96, &quotient)
	return result
}

func getAmountXDelta(sqrtRatioA, sqrtRatioB, liquidity *uint256.Int, roundUp bool) *uint256.Int {
	sa, sb := sqrtRatioA, sqrtRatioB
	if sa.Cmp(sb) > 0 {
		sa, sb = sb, sa
	}

	numerator1 := new(uint256.Int).Lsh(liquidity, 96)
	numerator2 := new(uint256.Int).Sub(sb, sa)

	if roundUp {
		var md uint256.Int
		big256.MulDivUp(&md, numerator1, numerator2, sb)
		return ceilDiv(&md, sa)
	}
	var md uint256.Int
	big256.MulDivDown(&md, numerator1, numerator2, sb)
	result := new(uint256.Int).Div(&md, sa)
	return result
}

func getAmountYDelta(sqrtRatioA, sqrtRatioB, liquidity *uint256.Int, roundUp bool) *uint256.Int {
	sa, sb := sqrtRatioA, sqrtRatioB
	if sa.Cmp(sb) > 0 {
		sa, sb = sb, sa
	}

	diff := new(uint256.Int).Sub(sb, sa)

	if roundUp {
		var result uint256.Int
		big256.MulDivUp(&result, liquidity, diff, q96)
		return &result
	}
	var result uint256.Int
	big256.MulDivDown(&result, liquidity, diff, q96)
	return &result
}

func ceilDiv(a, b *uint256.Int) *uint256.Int {
	q := new(uint256.Int).Div(a, b)
	var rem uint256.Int
	rem.Mod(a, b)
	if !rem.IsZero() {
		q.AddUint64(q, 1)
	}
	return q
}

func quoteXToY(params *PoolParams, dx *uint256.Int) *QuoteResult {
	zero := &QuoteResult{
		AmountOut:     new(uint256.Int),
		SqrtPriceNext: new(uint256.Int).Set(params.SqrtPriceX96),
		Fee:           new(uint256.Int),
	}

	cQ48 := concentrationQ48(params.FeeQ48, dx, params.ReserveX, params.ConcentrationK)
	if cQ48.Cmp(q48) >= 0 {
		return zero
	}

	cU64 := cQ48.Uint64()
	liquidity := computeLiquidity(params.SqrtPriceX96, cU64, params.ReserveX, params.ReserveY)

	oneMinusCQ48 := new(uint256.Int).Sub(q48, cQ48)
	sqrtOneMinusC := isqrt(oneMinusCQ48)
	var pBid uint256.Int
	big256.MulDivDown(&pBid, params.SqrtPriceX96, sqrtOneMinusC, q24)

	maxNetDx := getAmountXDelta(&pBid, params.SqrtPriceX96, liquidity, false)
	if dx.Cmp(maxNetDx) > 0 {
		return zero
	}

	pNext := getNextSqrtPriceFromAmountXRoundingUp(params.SqrtPriceX96, liquidity, dx)
	dy := getAmountYDelta(params.SqrtPriceX96, pNext, liquidity, false)

	var fee uint256.Int
	big256.MulDivDown(&fee, dy, uint256.NewInt(params.FeeQ48), q48)
	dyAfterFee := new(uint256.Int).Sub(dy, &fee)

	return &QuoteResult{
		AmountOut:     dyAfterFee,
		SqrtPriceNext: pNext,
		Fee:           new(uint256.Int).Set(&fee),
	}
}

func quoteYToX(params *PoolParams, dy *uint256.Int) *QuoteResult {
	zero := &QuoteResult{
		AmountOut:     new(uint256.Int),
		SqrtPriceNext: new(uint256.Int).Set(params.SqrtPriceX96),
		Fee:           new(uint256.Int),
	}

	cQ48 := concentrationQ48(params.FeeQ48, dy, params.ReserveY, params.ConcentrationK)
	if cQ48.Cmp(q48) >= 0 {
		return zero
	}

	cU64 := cQ48.Uint64()
	liquidity := computeLiquidity(params.SqrtPriceX96, cU64, params.ReserveX, params.ReserveY)

	oneMinusCQ48 := new(uint256.Int).Sub(q48, cQ48)
	sqrtOneMinusC := isqrt(oneMinusCQ48)
	var pAsk uint256.Int
	big256.MulDivDown(&pAsk, params.SqrtPriceX96, q24, sqrtOneMinusC)

	maxNetDy := getAmountYDelta(params.SqrtPriceX96, &pAsk, liquidity, false)
	if dy.Cmp(maxNetDy) > 0 {
		return zero
	}

	pNext := getNextSqrtPriceFromAmountYRoundingDown(params.SqrtPriceX96, liquidity, dy)
	dxOut := getAmountXDelta(params.SqrtPriceX96, pNext, liquidity, false)

	var fee uint256.Int
	big256.MulDivDown(&fee, dxOut, uint256.NewInt(params.FeeQ48), q48)
	dxAfterFee := new(uint256.Int).Sub(dxOut, &fee)

	return &QuoteResult{
		AmountOut:     dxAfterFee,
		SqrtPriceNext: pNext,
		Fee:           new(uint256.Int).Set(&fee),
	}
}

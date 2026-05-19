package brownfiv3

import (
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// pythToQ64 converts a Pyth price mantissa (decimal string) and exponent to Q64.
// Result = mantissa * Q64 / 10^(-expo), where expo is negative (e.g. -8).
func pythToQ64(mantissaStr string, expo int) *uint256.Int {
	var m uint256.Int
	if err := m.SetFromDecimal(mantissaStr); err != nil || m.Sign() == 0 {
		return new(uint256.Int)
	}
	denom := big256.TenPow(-expo) // -expo is positive
	var res uint256.Int
	big256.MulDivDown(&res, &m, q64, denom)
	return &res
}

// computeSwapPrices replicates factory.getSwapPrices() off-chain following §3 of the V3 design.
//
// price0, price1: Q64 Pyth dollar prices for token0, token1.
// conf0, conf1:   Q64 Pyth confidence intervals.
// ammPrice:       Q64 on-chain AMM relative price (quote-per-base); 0 if no valid pool.
// reserveBase18, reserveQuote18: 18-dec reserves of the base and quote tokens.
// isSell: true iff tokenOut is the base token.
//
// Returns priceIn, priceOut in Q64 dollar-price units (for AMM formula) and adjPrice
// in Q64 relative price (quote-per-base, for the γ-cutoff formula).
func computeSwapPrices(
	price0, price1, conf0, conf1 *uint256.Int,
	ammPrice *uint256.Int,
	reserveBase18, reserveQuote18 *uint256.Int,
	quoteTokenIndex uint8,
	pythWeight, fixS, compress, sSell, sBuy, sBound, disThreshold uint32,
	lambda uint64,
	fee uint32,
	isSell bool,
) (priceIn, priceOut, adjPrice *uint256.Int, err error) {
	// ── Step 1: identify base/quote Q64 dollar prices ─────────────────────
	pythPriceB, pythPriceQ := price0, price1
	confB, confQ := conf0, conf1
	if quoteTokenIndex == 0 { // token0=quote, token1=base
		pythPriceB, pythPriceQ = price1, price0
		confB, confQ = conf1, conf0
	}
	if pythPriceB.Sign() == 0 || pythPriceQ.Sign() == 0 {
		return nil, nil, nil, ErrZeroPythPrice
	}

	// ── Step 2: adjPrice = pythWeight*pythRelPrice + (1-pythWeight)*ammPrice ──
	// pythRelPrice = pythPriceB * Q64 / pythPriceQ  (quote-per-base in Q64)
	var pythRelPrice uint256.Int
	big256.MulDivDown(&pythRelPrice, pythPriceB, q64, pythPriceQ)

	var adj uint256.Int
	hasAmm := ammPrice != nil && ammPrice.Sign() > 0
	pw := uint64(pythWeight)
	prec := precisionU.Uint64()
	if !hasAmm || pw >= prec {
		adj.Set(&pythRelPrice)
	} else {
		var t1, t2, pwU, remU uint256.Int
		pwU.SetUint64(pw)
		remU.SetUint64(prec - pw)
		big256.MulDivDown(&t1, &pythRelPrice, &pwU, precisionU)
		big256.MulDivDown(&t2, ammPrice, &remU, precisionU)
		adj.Add(&t1, &t2)
	}
	if adj.Sign() == 0 {
		return nil, nil, nil, ErrZeroAdjPrice
	}

	// ── Step 3: OSpread ───────────────────────────────────────────────────
	var oSpread uint256.Int
	if hasAmm {
		// |pythRelPrice - ammPrice| * PRECISION / adjPrice
		var diff uint256.Int
		if pythRelPrice.Gt(ammPrice) {
			diff.Sub(&pythRelPrice, ammPrice)
		} else {
			diff.Sub(ammPrice, &pythRelPrice)
		}
		big256.MulDivDown(&oSpread, &diff, precisionU, &adj)
	} else {
		// Confidence-based spread:
		// upperPrice = (pythPriceB + confB) * Q64 / max(pythPriceQ - confQ, 1)
		// lowerPrice = max(pythPriceB - confB, 0) * Q64 / (pythPriceQ + confQ)
		// OSpread = (upperPrice - lowerPrice) * PRECISION / adjPrice
		var upperB, lowerB, lowerQ, upperQ uint256.Int
		upperB.Add(pythPriceB, confB)
		if pythPriceQ.Gt(confQ) {
			lowerQ.Sub(pythPriceQ, confQ)
		} else {
			lowerQ.SetUint64(1)
		}
		big256.MulDivDown(&upperB, &upperB, q64, &lowerQ)

		upperQ.Add(pythPriceQ, confQ)
		if pythPriceB.Gt(confB) {
			lowerB.Sub(pythPriceB, confB)
			big256.MulDivDown(&lowerB, &lowerB, q64, &upperQ)
		}

		if upperB.Gt(&lowerB) {
			upperB.Sub(&upperB, &lowerB)
		} else {
			upperB.Clear()
		}
		big256.MulDivDown(&oSpread, &upperB, precisionU, &adj)
	}

	// ── Step 4: disThreshold check ────────────────────────────────────────
	if disThreshold > 0 && oSpread.GtUint64(uint64(disThreshold)) {
		return nil, nil, nil, ErrSpreadExceedsThreshold
	}

	// ── Step 5: skewness S = (b*adj/Q64 - q) / (b*adj/Q64 + q) ──────────
	var bv uint256.Int // base value in quote units (18-dec)
	big256.MulDivDown(&bv, reserveBase18, &adj, q64)

	var skewNumer, skewDenom uint256.Int
	skewDenom.Add(&bv, reserveQuote18)

	// ── Step 6: skew factor SF (PRECISION units) ──────────────────────────
	// maxSF = fee*PRECISION/(2*PRECISION+fee) + sBound
	var maxSF, denom2P uint256.Int
	denom2P.AddUint64(denom2P.Mul(precisionU, big256.U2), uint64(fee))
	big256.MulDivDown(&maxSF, uint256.NewInt(uint64(fee)), precisionU, &denom2P)
	maxSF.AddUint64(&maxSF, uint64(sBound))

	heavyBase := bv.Gt(reserveQuote18)
	if heavyBase {
		skewNumer.Sub(&bv, reserveQuote18)
	} else {
		skewNumer.Sub(reserveQuote18, &bv)
	}
	// SF_raw = skewNumer * lambda / skewDenom  (S * lambda in PRECISION units)
	// sf in PRECISION units: S * lambda_Q64 * PRECISION / Q64
	// = skewNumer * lambda * PRECISION / (skewDenom * Q64)
	var sf, sfNum, sfDen uint256.Int
	sfNum.Mul(&skewNumer, uint256.NewInt(lambda))
	sfDen.Mul(&skewDenom, q64)
	if sfDen.Sign() > 0 {
		big256.MulDivDown(&sf, &sfNum, precisionU, &sfDen)
	}
	if sf.Gt(&maxSF) {
		sf.Set(&maxSF)
	}
	if !sf.Lt(precisionU) { // SF >= 1 → skewness disabled (safety guard §6.3)
		sf.Clear()
	}

	// ── Step 7: SkewPrice = adj * (PREC±SF) / (PREC∓SF) ─────────────────
	var skewNum, skewDen uint256.Int
	if heavyBase {
		skewNum.Sub(precisionU, &sf)
		skewDen.Add(precisionU, &sf)
	} else {
		skewNum.Add(precisionU, &sf)
		skewDen.Sub(precisionU, &sf)
	}
	var skewPrice uint256.Int
	big256.MulDivDown(&skewPrice, &adj, &skewNum, &skewDen)

	// ── Step 8: dynamic spread = compress * OSpread / PRECISION ──────────
	var dynSpread uint256.Int
	big256.MulDivDown(&dynSpread, &oSpread, uint256.NewInt(uint64(compress)), precisionU)

	// ── Step 9: pre-trade price ───────────────────────────────────────────
	var preTradePrice uint256.Int
	if isSell {
		// sellPrice = skewPrice * (PRECISION + fixS + dynSpread + sSell) / PRECISION
		var totalMul uint256.Int
		totalMul.SetUint64(uint64(fixS) + uint64(sSell))
		totalMul.Add(&totalMul, &dynSpread)
		totalMul.Add(&totalMul, precisionU)
		big256.MulDivDown(&preTradePrice, &skewPrice, &totalMul, precisionU)
	} else {
		// buyPrice = skewPrice * (PRECISION - fixS - dynSpread - sBuy) / PRECISION
		spreadSum := new(uint256.Int).SetUint64(uint64(fixS) + uint64(sBuy))
		spreadSum.Add(spreadSum, &dynSpread)
		if !spreadSum.Lt(precisionU) {
			return nil, nil, nil, ErrBuySpreadTooLarge
		}
		var totalMul uint256.Int
		totalMul.Sub(precisionU, spreadSum)
		big256.MulDivDown(&preTradePrice, &skewPrice, &totalMul, precisionU)
	}
	if preTradePrice.Sign() == 0 {
		return nil, nil, nil, ErrZeroPreTradePrice
	}

	// ── Step 10: convert to Q64 absolute dollar prices for AMM formula ────
	// SELL (tokenIn=quote, tokenOut=base):
	//   priceIn = pythPriceQ, priceOut = preTradePrice * pythPriceQ / Q64
	// BUY (tokenIn=base, tokenOut=quote):
	//   priceIn = pythPriceB, priceOut = pythPriceB * Q64 / preTradePrice
	var pIn, pOut uint256.Int
	if isSell {
		pIn.Set(pythPriceQ)
		big256.MulDivDown(&pOut, &preTradePrice, pythPriceQ, q64)
	} else {
		pIn.Set(pythPriceB)
		big256.MulDivDown(&pOut, pythPriceB, q64, &preTradePrice)
	}
	if pOut.Sign() == 0 {
		return nil, nil, nil, ErrZeroOutputPrice
	}

	return &pIn, &pOut, &adj, nil
}

// parseRawToDefaultDecimals normalises a raw token amount to 18-decimal space.
func parseRawToDefaultDecimals(amount *uint256.Int, decimals uint8) *uint256.Int {
	var res uint256.Int
	if decimals > parsedDecimals {
		res.Div(amount, big256.TenPow(decimals-parsedDecimals))
	} else {
		res.Mul(amount, big256.TenPow(parsedDecimals-decimals))
	}
	return &res
}

// parseDefaultDecimalsToRaw converts from 18-decimal space to token decimals (floor).
func parseDefaultDecimalsToRaw(amount *uint256.Int, decimals uint8) *uint256.Int {
	var res uint256.Int
	if decimals > parsedDecimals {
		res.Mul(amount, big256.TenPow(decimals-parsedDecimals))
	} else {
		res.Div(amount, big256.TenPow(parsedDecimals-decimals))
	}
	return &res
}

// parseDefaultDecimalsToRawUp converts from 18-decimal space to token decimals, rounding up on downscale.
func parseDefaultDecimalsToRawUp(amount *uint256.Int, decimals uint8) *uint256.Int {
	if decimals > parsedDecimals {
		var res uint256.Int
		return res.Mul(amount, big256.TenPow(decimals-parsedDecimals))
	} else if decimals < parsedDecimals {
		return big256.DivUp(amount, big256.TenPow(parsedDecimals-decimals))
	}
	return amount.Clone()
}

// amountInNoFeeToDefaultDecimals strips the fee from amountIn then normalises to 18 decimals.
// Mirrors BrownFiV3Pair's fee stripping order: floor in raw decimals first, then normalise.
func amountInNoFeeToDefaultDecimals(decimals uint8, amountIn *uint256.Int, fee uint32) *uint256.Int {
	// noFee = floor(amountIn * PRECISION / (PRECISION + fee))
	var denom, noFee uint256.Int
	denom.AddUint64(precisionU, uint64(fee))
	_ = v3Utils.MulDivV2(amountIn, precisionU, &denom, &noFee, nil)
	return parseRawToDefaultDecimals(&noFee, decimals)
}

// computeLeftNumerator = priceOut*reserveOut + priceIn*pseudoIn  (Q64 × 18-dec units)
func computeLeftNumerator(pseudoIn, priceIn, priceOut, reserveOut *uint256.Int) *uint256.Int {
	var res, tmp uint256.Int
	res.Mul(priceOut, reserveOut)
	tmp.Mul(priceIn, pseudoIn)
	return res.Add(&res, &tmp)
}

// computeLeftSqrt = abs(pseudoIn*priceIn/Q64 - reserveOut*priceOut/Q64)²  ((18-dec)² units)
func computeLeftSqrt(pseudoIn, priceIn, priceOut, reserveOut *uint256.Int) *uint256.Int {
	var a, b uint256.Int
	_ = v3Utils.MulDivV2(pseudoIn, priceIn, q64, &a, nil)
	_ = v3Utils.MulDivV2(reserveOut, priceOut, q64, &b, nil)
	if a.Gt(&b) {
		a.Sub(&a, &b)
	} else {
		a.Sub(&b, &a)
	}
	return a.Mul(&a, &a)
}

// computeRightSqrt = (priceIn*priceOut*kappa/Q128) * (reserveOut*pseudoIn*2/Q64)
func computeRightSqrt(pseudoIn, priceIn, priceOut, reserveOut, kappa *uint256.Int) *uint256.Int {
	var step1, combined uint256.Int
	step1.Mul(priceIn, priceOut)
	_ = v3Utils.MulDivV2(&step1, kappa, q128, &step1, nil)
	combined.Mul(reserveOut, pseudoIn)
	_ = v3Utils.MulDivV2(&combined, big256.U2, q64, &combined, nil)
	return step1.Mul(&step1, &combined)
}

// getAmountOutWithoutCutoff computes the raw quadratic amountOut without applying the §4.3 cutoff.
func getAmountOutWithoutCutoff(
	inDec, outDec uint8,
	amountIn, reserveOut, priceIn, priceOut, kappa *uint256.Int,
	fee uint32,
) (*uint256.Int, error) {
	if amountIn.Sign() == 0 {
		return nil, ErrInsufficientInputAmount
	}
	if reserveOut.Sign() == 0 {
		return nil, ErrInsufficientLiquidity
	}

	parsedReserveOut := parseRawToDefaultDecimals(reserveOut, outDec)
	pseudoIn := amountInNoFeeToDefaultDecimals(inDec, amountIn, fee)

	var amountOut uint256.Int

	if kappa.Cmp(q64x2) == 0 {
		// Uniswap-V2-like: mulDiv(reserveOut*pseudoIn, priceIn, priceOut*reserveOut + pseudoIn*priceIn)
		var num, denom, tmp uint256.Int
		num.Mul(parsedReserveOut, pseudoIn)
		tmp.Mul(priceOut, parsedReserveOut)
		denom.Mul(pseudoIn, priceIn)
		denom.Add(&tmp, &denom)
		_ = v3Utils.MulDivV2(&num, priceIn, &denom, &amountOut, nil)
	} else {
		leftNum := computeLeftNumerator(pseudoIn, priceIn, priceOut, parsedReserveOut)
		leftSqrt := computeLeftSqrt(pseudoIn, priceIn, priceOut, parsedReserveOut)
		rightSqrt := computeRightSqrt(pseudoIn, priceIn, priceOut, parsedReserveOut, kappa)

		var denom uint256.Int
		_ = v3Utils.MulDivV2(priceOut, denom.Sub(q64x2, kappa), q64, &denom, nil)

		radicand := leftSqrt.Add(leftSqrt, rightSqrt)

		var sqrtBase, sqrtSq uint256.Int
		sqrtBase.Sqrt(radicand)
		// Round up: if sqrtBase² < radicand, sqrtBase++
		if sqrtSq.Mul(&sqrtBase, &sqrtBase).Lt(radicand) {
			sqrtBase.AddUint64(&sqrtBase, 1)
		}

		// sqrtTerm = Q64 * sqrtBase
		sqrtTerm := sqrtBase.Mul(q64, &sqrtBase)

		if leftNum.Lt(sqrtTerm) {
			return nil, ErrMathUnderflow
		}
		if denom.Sign() == 0 {
			return nil, ErrZeroDenominator
		}
		amountOut.Div(leftNum.Sub(leftNum, sqrtTerm), &denom)
	}

	// Under-quote adjustment: m = (amountOut >> 64) + 2; amountOut = max(amountOut - m, 0)
	var m uint256.Int
	m.Rsh(&amountOut, 64)
	m.AddUint64(&m, 2)
	if amountOut.Gt(&m) {
		amountOut.Sub(&amountOut, &m)
	} else {
		amountOut.Clear()
	}

	// Guard: amountOut must be strictly less than reserveOut (on-chain: require(postBase > 0) / require(postQuote > 0)).
	if !amountOut.Lt(parsedReserveOut) {
		return nil, ErrInsufficientLiquidity
	}

	rawOut := parseDefaultDecimalsToRaw(&amountOut, outDec)
	if rawOut.Sign() == 0 {
		return nil, ErrZeroOutputAmount
	}
	return rawOut, nil
}

// getAmountInWithoutCutoff computes the required amountIn for a desired amountOut without the §4.3 cutoff.
// isSell must match the swap direction so the penalty formula aligns with checkInventoryBaseQuote:
// SELL uses basePriceQ64/Q64 scaling; BUY values quote at Q64 (= $1, no priceOut factor).
func getAmountInWithoutCutoff(
	inDec, outDec uint8,
	amountOut, reserveOut, priceIn, priceOut, kappa *uint256.Int,
	fee uint32,
	isSell bool,
) (*uint256.Int, error) {
	if amountOut.Sign() == 0 {
		return nil, ErrInsufficientOutputAmount
	}
	if reserveOut.Sign() == 0 {
		return nil, ErrInsufficientLiquidity
	}

	parsedAmountOut := parseRawToDefaultDecimals(amountOut, outDec)
	parsedReserveOut := parseRawToDefaultDecimals(reserveOut, outDec)

	if !parsedAmountOut.Lt(parsedReserveOut) {
		return nil, ErrInsufficientLiquidity
	}

	// scaled = ceil(kappa * parsedAmountOut / ((parsedReserveOut - parsedAmountOut) * 2))
	var denom uint256.Int
	denom.Sub(parsedReserveOut, parsedAmountOut)
	denom.Mul(&denom, big256.U2)
	var scaled uint256.Int
	big256.MulDivUp(&scaled, kappa, parsedAmountOut, &denom)

	// Build num to satisfy the checkInventoryBaseQuote invariant exactly:
	//   SELL: invariant values output (base) at priceOut/Q64 → penalty has priceOut/Q64 factor
	//   BUY:  invariant values output (quote) at Q64 ($1)     → penalty is scaled*quoteOut only
	var num uint256.Int
	if isSell {
		var scaledAmt, penalty uint256.Int
		scaledAmt.Mul(&scaled, parsedAmountOut)
		big256.MulDivUp(&penalty, priceOut, &scaledAmt, q64)
		num.Mul(priceOut, parsedAmountOut)
		num.Add(&num, &penalty)
	} else {
		var penalty uint256.Int
		penalty.Mul(&scaled, parsedAmountOut)
		num.Mul(q64, parsedAmountOut)
		num.Add(&num, &penalty)
	}
	pseudoIn := big256.DivUp(&num, priceIn)

	// amountIn18 = ceil(pseudoIn * (PRECISION + fee) / PRECISION)
	var precPlusFee uint256.Int
	precPlusFee.AddUint64(precisionU, uint64(fee))
	pseudoIn.Mul(pseudoIn, &precPlusFee)
	amountIn18 := big256.DivUp(pseudoIn, precisionU)

	return parseDefaultDecimalsToRawUp(amountIn18, inDec), nil
}

// getAmountOutCutoff returns the §4.3 gamma cap on amountOut given amountIn.
// Returns max uint256 when gamma >= PRECISION or adjPrice is zero (cutoff disabled).
// On-chain: `if (gamma >= PRECISION) return (amount0Out, amount1Out)` in _applyGammaCutoff.
func getAmountOutCutoff(
	inDec, outDec uint8,
	amountIn, reserveIn, reserveOut *uint256.Int,
	adjPrice *uint256.Int,
	gamma uint32,
	fee uint32,
	isSell bool,
) (*uint256.Int, error) {
	if uint64(gamma) >= precisionU.Uint64() || adjPrice.Sign() == 0 {
		return big256.UMax.Clone(), nil
	}

	parsedReserveIn := parseRawToDefaultDecimals(reserveIn, inDec)
	parsedReserveOut := parseRawToDefaultDecimals(reserveOut, outDec)
	pseudoIn := amountInNoFeeToDefaultDecimals(inDec, amountIn, fee)

	var gNum, gDen, inputPlusIn, subTerm18 uint256.Int
	gNum.SubUint64(precisionU, uint64(gamma))  // PRECISION - gamma
	gDen.AddUint64(precisionU, uint64(gamma))  // PRECISION + gamma
	inputPlusIn.Add(parsedReserveIn, pseudoIn) // q+dq (SELL) or b+db (BUY)

	if isSell {
		// sub = ceil((PRECISION-gamma)*(q+dq)*Q64 / ((PRECISION+gamma)*adjPrice))
		// On-chain uses mulDivRoundingUp — ceiling rounds sub up, cutoff down (conservative).
		var numer, denominator uint256.Int
		numer.Mul(&gNum, &inputPlusIn)
		denominator.Mul(&gDen, adjPrice)
		big256.MulDivUp(&subTerm18, &numer, q64, &denominator)
	} else {
		// sub = ceil((PRECISION-gamma)*(b+db)*adjPrice / ((PRECISION+gamma)*Q64))
		var numer, denominator uint256.Int
		numer.Mul(&gNum, &inputPlusIn)
		denominator.Mul(&gDen, q64)
		big256.MulDivUp(&subTerm18, &numer, adjPrice, &denominator)
	}

	var cutoff18 uint256.Int
	if parsedReserveOut.Gt(&subTerm18) {
		cutoff18.Sub(parsedReserveOut, &subTerm18)
	}
	return parseDefaultDecimalsToRaw(&cutoff18, outDec), nil
}

// maxOutClosedForm computes the closed-form maximum amountOUT (in 18-dec) before the §4.3 cutoff fires.
//
// bc is always positive in both directions (X18 > Ymu is guarded above), so we track |aQ64| and
// compute |num| = bc - sqrtTerm directly, avoiding signed arithmetic entirely.
//
// Returns (nil, nil) when pool is past gamma; returns (value, nil) otherwise.
// Callers must handle the gamma==0 || adjPr==0 disabled case before calling.
func maxOutClosedForm(X18, Y18, P, K *uint256.Int, gamma uint32, adjPr *uint256.Int, isSell bool) (*uint256.Int, error) {
	if K.Gt(q64x2) {
		// K > 2*Q64: aQ64 sign flips, formula inapplicable; treat as underflow.
		return nil, ErrMathUnderflow
	}

	var gammaU, gN, gDen uint256.Int
	gammaU.SetUint64(uint64(gamma))
	gN.Sub(precisionU, &gammaU)
	gDen.Add(precisionU, &gammaU)

	if isSell {
		// gDxAdj = (PRECISION+gamma) * adjPr
		var gDxAdj uint256.Int
		gDxAdj.Mul(&gDen, adjPr)

		// gnY18 = gN * Y18
		var gnY18, Ymu uint256.Int
		gnY18.Mul(&gN, Y18)
		big256.MulDivDown(&Ymu, &gnY18, q64, &gDxAdj)
		if !X18.Gt(&Ymu) {
			return nil, nil // pool past gamma
		}
		var diff uint256.Int
		diff.Sub(X18, &Ymu)

		var twoQ64MinusK uint256.Int
		twoQ64MinusK.Sub(q64x2, K)

		// rP_Q64 = gN * P * Q64 / gDxAdj  (reused for |aQ64| and bc)
		var rP_Q64, gnP uint256.Int
		gnP.Mul(&gN, P)
		big256.MulDivDown(&rP_Q64, &gnP, q64, &gDxAdj)

		// |aQ64| = 2*Q64 + rP_Q64 * twoQ64MinusK / Q64
		var absAQ64 uint256.Int
		big256.MulDivDown(&absAQ64, &rP_Q64, &twoQ64MinusK, q64)
		absAQ64.Add(&absAQ64, q64x2)

		// bc = 2 * (2*X18 + rP_Q64*X18/Q64 - Ymu)  — always positive since X18 > Ymu
		var rPX, bc uint256.Int
		big256.MulDivDown(&rPX, &rP_Q64, X18, q64)
		bc.Add(X18, X18)
		bc.Add(&bc, &rPX)
		bc.Sub(&bc, &Ymu)
		bc.Mul(&bc, big256.U2)

		// Discriminant: PY = P*X18/Q64; sumPYX = PY+Y18; t1disc = (sumPYX² * gN * Q64 / gDxAdj)²/gDxAdj² (two-step)
		var PY, sumPYX, sumSq uint256.Int
		big256.MulDivDown(&PY, P, X18, q64)
		sumPYX.Add(&PY, Y18)
		sumSq.Mul(&sumPYX, &sumPYX)

		var gnQ, rSumSq, t1disc uint256.Int
		gnQ.Mul(&gN, q64)
		big256.MulDivDown(&rSumSq, &sumSq, &gnQ, &gDxAdj)
		big256.MulDivDown(&t1disc, &rSumSq, &gnQ, &gDxAdj)

		var KPQ64, KPX, kImp, t2disc uint256.Int
		big256.MulDivDown(&KPQ64, K, P, q64)
		big256.MulDivDown(&KPX, &KPQ64, X18, q64)
		kImp.Mul(&KPX, &diff)
		kImp.Mul(&kImp, big256.U2)
		big256.MulDivDown(&t2disc, &kImp, &gnQ, &gDxAdj)

		var disc, sqrtDelta, sqrtSq uint256.Int
		disc.Add(&t1disc, &t2disc)
		disc.Mul(&disc, big256.U4)
		sqrtDelta.Sqrt(&disc)
		if sqrtSq.Mul(&sqrtDelta, &sqrtDelta).Lt(&disc) {
			sqrtDelta.AddUint64(&sqrtDelta, 1)
		}

		// |num| = bc - sqrtDelta; both negative in signed form → positive quotient
		if sqrtDelta.Gt(&bc) {
			return nil, ErrMathUnderflow
		}
		var absNum, den, result uint256.Int
		absNum.Sub(&bc, &sqrtDelta)
		den.Mul(&absAQ64, big256.U2)
		big256.MulDivDown(&result, &absNum, q64, &den)
		return &result, nil

	} else {
		// BUY: scale = gN*adjPr, denom = gDen*Q64
		var gDxQ64 uint256.Int
		gDxQ64.Mul(&gDen, q64)

		var gnAdjPr, gnY18, Ymu uint256.Int
		gnAdjPr.Mul(&gN, adjPr)
		gnY18.Mul(&gN, Y18)
		big256.MulDivDown(&Ymu, &gnY18, adjPr, &gDxQ64)
		if !X18.Gt(&Ymu) {
			return nil, nil
		}
		var diff uint256.Int
		diff.Sub(X18, &Ymu)

		var twoQ64MinusK uint256.Int
		twoQ64MinusK.Sub(q64x2, K)

		// |aQ64| = 2*P + gN*adjPr*twoQ64MinusK/gDxQ64
		var betaTerm, absAQ64 uint256.Int
		big256.MulDivDown(&betaTerm, &gnAdjPr, &twoQ64MinusK, &gDxQ64)
		absAQ64.Add(P, P)
		absAQ64.Add(&absAQ64, &betaTerm)

		// betaQ64 = gN*adjPr/gDen;  betaP_Q64 = betaQ64*P/Q64
		var betaQ64, betaP_Q64 uint256.Int
		betaQ64.Div(&gnAdjPr, &gDen)
		big256.MulDivDown(&betaP_Q64, &betaQ64, P, q64)

		// bc = 4*P*X18 + 2*X18*betaQ64 - 2*Y18*betaP_Q64
		// If t3 > t1+t2, bc would be negative → num > 0 → underflow
		var t1, t2, t3, bc uint256.Int
		t1.Mul(P, X18)
		t1.Mul(&t1, big256.U4)
		t2.Mul(X18, &betaQ64)
		t2.Mul(&t2, big256.U2)
		t3.Mul(Y18, &betaP_Q64)
		t3.Mul(&t3, big256.U2)
		bc.Add(&t1, &t2)
		if t3.Gt(&bc) {
			return nil, ErrMathUnderflow
		}
		bc.Sub(&bc, &t3)

		// Discriminant: PY = P*Y18/Q64; sumPYX = PY+X18
		var PY, sumPYX, sumSq uint256.Int
		big256.MulDivDown(&PY, P, Y18, q64)
		sumPYX.Add(&PY, X18)
		sumSq.Mul(&sumPYX, &sumPYX)

		var rSumSq, t1disc uint256.Int
		big256.MulDivDown(&rSumSq, &sumSq, &gnAdjPr, &gDxQ64)
		big256.MulDivDown(&t1disc, &rSumSq, &gnAdjPr, &gDxQ64)

		var KPQ64, KPX, kImp, t2disc uint256.Int
		big256.MulDivDown(&KPQ64, K, P, q64)
		big256.MulDivDown(&KPX, &KPQ64, X18, q64)
		kImp.Mul(&KPX, &diff)
		kImp.Mul(&kImp, big256.U2)
		big256.MulDivDown(&t2disc, &kImp, &gnAdjPr, &gDxQ64)

		var disc, sqrtDelta, sqrtSq uint256.Int
		disc.Add(&t1disc, &t2disc)
		disc.Mul(&disc, big256.U4)
		sqrtDelta.Sqrt(&disc)
		if sqrtSq.Mul(&sqrtDelta, &sqrtDelta).Lt(&disc) {
			sqrtDelta.AddUint64(&sqrtDelta, 1)
		}

		// BUY: num = -bc + sqrtDelta*Q64 (bcScaled); |num| = bc - sqrtDelta*Q64
		var sqrtTerm uint256.Int
		sqrtTerm.Mul(&sqrtDelta, q64)
		if sqrtTerm.Gt(&bc) {
			return nil, ErrMathUnderflow
		}
		var absNum, den, result uint256.Int
		absNum.Sub(&bc, &sqrtTerm)
		den.Mul(&absAQ64, big256.U2)
		result.Div(&absNum, &den)
		return &result, nil
	}
}

// maxAmountOutRaw returns the closed-form max amountOUT in raw output-token decimals.
// Returns nil + ErrPoolPastGamma when the pool is already past gamma; max uint256 when cutoff disabled.
func maxAmountOutRaw(
	inDec, outDec uint8,
	reserveIn, reserveOut, priceIn, priceOut, adjPrice, kappa *uint256.Int,
	gamma uint32,
	isSell bool,
) (*uint256.Int, error) {
	if uint64(gamma) >= precisionU.Uint64() || adjPrice.Sign() == 0 {
		return big256.UMax.Clone(), nil
	}

	parsedReserveOut := parseRawToDefaultDecimals(reserveOut, outDec)
	parsedReserveIn := parseRawToDefaultDecimals(reserveIn, inDec)

	// P = swap price in Q64, quote-per-base
	// SELL (OUT=base): P = priceOut/priceIn; BUY (OUT=quote): P = priceIn/priceOut
	var P uint256.Int
	if isSell {
		_ = v3Utils.MulDivV2(priceOut, q64, priceIn, &P, nil)
	} else {
		_ = v3Utils.MulDivV2(priceIn, q64, priceOut, &P, nil)
	}

	outStar18, err := maxOutClosedForm(parsedReserveOut, parsedReserveIn, &P, kappa, gamma, adjPrice, isSell)
	if err != nil {
		return nil, err
	}
	if outStar18 == nil {
		return nil, ErrPoolPastGamma
	}

	return parseDefaultDecimalsToRaw(outStar18, outDec), nil
}

// checkInventoryBaseQuote mirrors the on-chain BrownFiV3Pair._checkInventoryBaseQuote.
// All amounts are in raw token decimals; the function normalises internally to 18-dec.
//
// isSell=true → tokenIn=quote, tokenOut=base.
// adjPrice = basePriceQ64 (quote-per-base, Q64).
func checkInventoryBaseQuote(
	inDec, outDec uint8,
	amountIn, amountOut, reserveIn, reserveOut *uint256.Int,
	basePriceQ64, kappa *uint256.Int,
	fee uint32,
	isSell bool,
) error {
	pseudoIn18 := amountInNoFeeToDefaultDecimals(inDec, amountIn, fee)
	amountOut18 := parseRawToDefaultDecimals(amountOut, outDec)
	reserveIn18 := parseRawToDefaultDecimals(reserveIn, inDec)
	reserveOut18 := parseRawToDefaultDecimals(reserveOut, outDec)

	var preBase18, preQuote18, postBase18, postQuote18, baseOut18, quoteOut18 uint256.Int
	if isSell {
		// tokenIn=quote, tokenOut=base
		preBase18.Set(reserveOut18)
		preQuote18.Set(reserveIn18)
		baseOut18.Set(amountOut18)
		// quoteOut18 = 0
		if !preBase18.Gt(&baseOut18) {
			return ErrInsufficientLiquidity
		}
		postBase18.Sub(&preBase18, &baseOut18)
		postQuote18.Add(&preQuote18, pseudoIn18)
		if postBase18.Sign() <= 0 {
			return ErrInvalidInventory
		}
	} else {
		// tokenIn=base, tokenOut=quote
		preBase18.Set(reserveIn18)
		preQuote18.Set(reserveOut18)
		// baseOut18 = 0
		quoteOut18.Set(amountOut18)
		if !preQuote18.Gt(&quoteOut18) {
			return ErrInsufficientLiquidity
		}
		postBase18.Add(&preBase18, pseudoIn18)
		postQuote18.Sub(&preQuote18, &quoteOut18)
		if postQuote18.Sign() <= 0 {
			return ErrInvalidInventory
		}
	}

	// left = adjPrice * postBase + Q64 * postQuote
	// right = adjPrice * preBase  + Q64 * preQuote
	var left, right, t1, t2 uint256.Int
	t1.Mul(basePriceQ64, &postBase18)
	t2.Mul(q64, &postQuote18)
	left.Add(&t1, &t2)

	t1.Mul(basePriceQ64, &preBase18)
	t2.Mul(q64, &preQuote18)
	right.Add(&t1, &t2)

	var penalty, denom, scaledOut uint256.Int
	if isSell {
		// scaled = mulDivUp(kappa, baseOut, postBase*2)
		// penalty = mulDivUp(basePriceQ64, scaled*baseOut, Q64)
		denom.Mul(&postBase18, big256.U2)
		big256.MulDivUp(&scaledOut, kappa, &baseOut18, &denom)
		scaledOut.Mul(&scaledOut, &baseOut18)
		big256.MulDivUp(&penalty, basePriceQ64, &scaledOut, q64)
	} else {
		// scaled = mulDivUp(kappa, quoteOut, postQuote*2)
		// penalty = scaled * quoteOut  (exact — no rounding, per contract comment)
		denom.Mul(&postQuote18, big256.U2)
		big256.MulDivUp(&scaledOut, kappa, &quoteOut18, &denom)
		penalty.Mul(&scaledOut, &quoteOut18)
	}

	// require left >= right + penalty
	right.Add(&right, &penalty)
	if left.Lt(&right) {
		return ErrInvalidInventory
	}
	return nil
}

// calcAmountOut computes amountOut with §4.3 cutoff protection.
func calcAmountOut(
	inDec, outDec uint8,
	amountIn, reserveIn, reserveOut, priceIn, priceOut, adjPrice, kappa *uint256.Int,
	gamma uint32, fee uint32, isSell bool,
) (*uint256.Int, error) {
	amountOut, err := getAmountOutWithoutCutoff(inDec, outDec, amountIn, reserveOut, priceIn, priceOut, kappa, fee)
	if err != nil {
		return nil, err
	}
	cutoff, err := getAmountOutCutoff(inDec, outDec, amountIn, reserveIn, reserveOut, adjPrice, gamma, fee, isSell)
	if err != nil {
		return nil, err
	}
	if amountOut.Gt(cutoff) {
		return nil, ErrCutoffInputLimitReached
	}
	// basePriceQ64 = dollar price of base token: priceOut when selling base, priceIn when buying base.
	// On-chain: basePriceQ64 = token0IsBase ? sPrice0 : sPrice1.
	basePriceQ64 := priceOut
	if !isSell {
		basePriceQ64 = priceIn
	}
	if err = checkInventoryBaseQuote(inDec, outDec, amountIn, amountOut, reserveIn, reserveOut, basePriceQ64, kappa, fee, isSell); err != nil {
		return nil, err
	}
	// _checkPostTradeSkew is intentionally omitted: the gamma cutoff above enforces the same
	// |S_post| <= gamma/PRECISION bound. A separate skew check would use our off-chain adjPrice,
	// which can diverge from the on-chain value due to Pyth staleness, making it an unreliable gate.
	return amountOut, nil
}

// calcAmountIn computes amountIn with §4.3 cutoff check (PDF §8.2 single-pass).
func calcAmountIn(
	inDec, outDec uint8,
	amountOut, reserveIn, reserveOut, priceIn, priceOut, adjPrice, kappa *uint256.Int,
	gamma uint32, fee uint32, isSell bool,
) (*uint256.Int, error) {
	outStarRaw, err := maxAmountOutRaw(inDec, outDec, reserveIn, reserveOut, priceIn, priceOut, adjPrice, kappa, gamma, isSell)
	if err != nil {
		return nil, err
	}
	if amountOut.Gt(outStarRaw) {
		return nil, ErrCutoffLimitReached
	}
	amountIn, err := getAmountInWithoutCutoff(inDec, outDec, amountOut, reserveOut, priceIn, priceOut, kappa, fee, isSell)
	if err != nil {
		return nil, err
	}
	basePriceQ64 := priceOut
	if !isSell {
		basePriceQ64 = priceIn
	}
	if err = checkInventoryBaseQuote(inDec, outDec, amountIn, amountOut, reserveIn, reserveOut, basePriceQ64, kappa, fee, isSell); err != nil {
		return nil, err
	}
	return amountIn, nil
}

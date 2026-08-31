package umbraedlmm

import "github.com/holiman/uint256"

// Shared constants. These are read-only after init; never mutate them in place.
var (
	e18     = uint256.NewInt(1_000_000_000_000_000_000)
	e10     = uint256.NewInt(10_000_000_000)
	uMaxU   = new(uint256.Int).Not(uint256.NewInt(0)) // 2^256 - 1, the overflow cap value
	uBP     = uint256.NewInt(basisPoints)
	uMaxFee = uint256.NewInt(maxFeeBps)
)

// pow10 returns 10^n as a uint256.
func pow10(n uint8) *uint256.Int {
	return new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(uint64(n)))
}

// getNormalizedPriceFromId computes (1 + binStep/10000)^(binId-ACTIVE_BIN) in 1e18 fixed point,
// mirroring the deployed BinHelper.getNormalizedPriceFromId exactly: plain 1e18
// exponentiation-by-squaring (base = numerator*1e18/denominator; result = result*base/1e18;
// base = base*base/1e18), including the saturate-on-overflow guard. Sentinels: 0 (underflow) and
// 2^256-1 (overflow); isNormalizedPriceRepresentable rejects both.
func getNormalizedPriceFromId(binID uint32, binStep uint16) *uint256.Int {
	exponent := int64(binID) - int64(activeBinID)
	if exponent == 0 {
		return new(uint256.Int).Set(e18)
	}

	absExp := uint64(exponent)
	if exponent < 0 {
		absExp = uint64(-exponent)
	}
	var numerator, denominator uint64
	if exponent > 0 {
		numerator, denominator = basisPoints+uint64(binStep), basisPoints
	} else {
		numerator, denominator = basisPoints, basisPoints+uint64(binStep)
	}

	result := new(uint256.Int).Set(e18)
	base := new(uint256.Int).Mul(uint256.NewInt(numerator), e18)
	base.Div(base, uint256.NewInt(denominator))

	for absExp > 0 {
		if absExp&1 == 1 {
			result = mulDiv1e18Capped(result, base)
		}
		base = mulDiv1e18Capped(base, base)
		absExp >>= 1
	}
	return result
}

// mulDiv1e18Capped returns floor(a*b/1e18), saturating to 2^256-1 on overflow (matching the
// deployed `result > type(uint256).max / base` guard). For bins within the tradable range this
// never saturates, but it is replicated for exactness.
func mulDiv1e18Capped(a, b *uint256.Int) *uint256.Int {
	if !b.IsZero() {
		var lim uint256.Int
		lim.Div(uMaxU, b)
		if a.Cmp(&lim) > 0 {
			return new(uint256.Int).Set(uMaxU)
		}
	}
	r := new(uint256.Int).Mul(a, b)
	return r.Div(r, e18)
}

// isNormalizedPriceRepresentable mirrors BinHelper.isNormalizedPriceRepresentable: both saturation
// sentinels fail closed (V2 rejects unrepresentable bins in BOTH swap directions, #126/#121).
func isNormalizedPriceRepresentable(price *uint256.Int) bool {
	return !price.IsZero() && price.Cmp(uMaxU) != 0
}

// mulDivFloor mirrors SwapCalculator._mulDivFloor: a*(b/d) + (a*(b%d))/d. The intermediates are
// NOT 512-bit — a uint256 overflow reverts on-chain, so it errors here rather than wrapping.
func mulDivFloor(a, b, d *uint256.Int) (*uint256.Int, error) {
	q := new(uint256.Int).Div(b, d)
	r := new(uint256.Int).Mod(b, d)
	t1, over := new(uint256.Int).MulOverflow(a, q)
	if over {
		return nil, ErrMathOverflow
	}
	t2, over := new(uint256.Int).MulOverflow(a, r)
	if over {
		return nil, ErrMathOverflow
	}
	t2.Div(t2, d)
	res, over := new(uint256.Int).AddOverflow(t1, t2)
	if over {
		return nil, ErrMathOverflow
	}
	return res, nil
}

// mulDivCeil mirrors SwapCalculator._mulDivCeil: a*(b/d) + (a*(b%d) + d - 1)/d, erroring where the
// checked Solidity arithmetic would revert.
func mulDivCeil(a, b, d *uint256.Int) (*uint256.Int, error) {
	q := new(uint256.Int).Div(b, d)
	r := new(uint256.Int).Mod(b, d)
	t1, over := new(uint256.Int).MulOverflow(a, q)
	if over {
		return nil, ErrMathOverflow
	}
	t2, over := new(uint256.Int).MulOverflow(a, r)
	if over {
		return nil, ErrMathOverflow
	}
	t2, over = t2.AddOverflow(t2, d)
	if over {
		return nil, ErrMathOverflow
	}
	t2.Sub(t2, uint256.NewInt(1))
	t2.Div(t2, d)
	res, over := new(uint256.Int).AddOverflow(t1, t2)
	if over {
		return nil, ErrMathOverflow
	}
	return res, nil
}

// calculateDynamicFee mirrors the deployed FeeHelper.calculateDynamicFee: baseFee + quadratic
// variable fee, total capped at MAX_FEE. The variableFeeCap clamp is UNCONDITIONAL in V2 — a zero
// cap pins the variable part to zero (#147), it does not mean "no cap". Returns the fee rate in
// basis points.
func calculateDynamicFee(baseFactor, variableFeeControl uint16, volatility uint64, binStep, variableFeeCap uint16) *uint256.Int {
	totalFee := uint256.NewInt(uint64(baseFactor))
	if variableFeeControl != 0 {
		prod := new(uint256.Int).Mul(uint256.NewInt(volatility), uint256.NewInt(uint64(binStep)))
		prod.Mul(prod, prod)
		prod.Mul(prod, uint256.NewInt(uint64(variableFeeControl)))
		prod.Div(prod, e10)
		feeCap := uint256.NewInt(uint64(variableFeeCap))
		if prod.Cmp(feeCap) > 0 {
			prod = feeCap
		}
		totalFee.Add(totalFee, prod)
	}
	if totalFee.Cmp(uMaxFee) > 0 {
		return new(uint256.Int).Set(uMaxFee)
	}
	return totalFee
}

// applyVolatilityDecay mirrors FeeHelper.applyVolatilityDecay: linear decay over decayPeriod, with
// a floor of 1 to avoid premature rounding to 0.
func applyVolatilityDecay(volatility uint64, timeDelta, decayPeriod uint64) uint64 {
	if decayPeriod == 0 || timeDelta >= decayPeriod {
		return 0
	}
	decayed := volatility * (decayPeriod - timeDelta) / decayPeriod
	if decayed == 0 && volatility > 0 {
		return 1
	}
	return decayed
}

// getDecayedVolatility mirrors FeeHelper.getDecayedVolatility: the accumulator holds constant
// inside the filter window and decays linearly only after it lapses.
func getDecayedVolatility(volatility, lastUpdate, filterPeriod, decayPeriod, now uint64) uint64 {
	if now <= lastUpdate {
		return volatility
	}
	delta := now - lastUpdate
	if delta < filterPeriod {
		return volatility
	}
	return applyVolatilityDecay(volatility, delta, decayPeriod)
}

// getFeeAmountFrom mirrors the deployed FeeHelper.getFeeAmountFrom: a CEILING division with
// denominator (10000 + totalFee), clamped so the fee never consumes the whole amount. (V1 floored;
// the V2 ceil + clamp are both load-bearing for wei-exactness.)
func getFeeAmountFrom(amount, totalFee *uint256.Int) (*uint256.Int, error) {
	if amount.IsZero() || totalFee.IsZero() {
		return uint256.NewInt(0), nil
	}
	den := new(uint256.Int).Add(uBP, totalFee)
	num, over := new(uint256.Int).MulOverflow(amount, totalFee)
	if over {
		return nil, ErrMathOverflow
	}
	num, over = num.AddOverflow(num, den)
	if over {
		return nil, ErrMathOverflow
	}
	num.Sub(num, uint256.NewInt(1))
	fee := num.Div(num, den)
	if fee.Cmp(amount) >= 0 {
		fee = new(uint256.Int).Sub(amount, uint256.NewInt(1))
	}
	return fee, nil
}

// calculateSwapWithinBin mirrors SwapCalculator.calculateSwapWithinBin. All amounts crossing this
// boundary are 18-decimal normalized; the price math runs in native units with the decimal
// conversion folded into priceDenominator = scaleY * 10^decimalsX (identical in both directions).
// Output floors, input-consumed-on-depletion ceils (against the swapper). A zero-output bin is
// skipped, not charged (H-02/#203): returns (0, 0).
func calculateSwapWithinBin(
	binReserveOut, amountInNormalized, normPrice, priceDenominator, scaleIn, scaleOut *uint256.Int,
	swapForY bool,
) (amountOutNormalized, amountInConsumedNormalized *uint256.Int, err error) {
	amountInNative := new(uint256.Int).Div(amountInNormalized, scaleIn)

	var maxOutNative *uint256.Int
	if swapForY {
		maxOutNative, err = mulDivFloor(amountInNative, normPrice, priceDenominator)
	} else {
		maxOutNative, err = mulDivFloor(amountInNative, priceDenominator, normPrice)
	}
	if err != nil {
		return nil, nil, err
	}
	maxOut, over := new(uint256.Int).MulOverflow(maxOutNative, scaleOut)
	if over {
		return nil, nil, ErrMathOverflow
	}

	if maxOut.IsZero() {
		return uint256.NewInt(0), uint256.NewInt(0), nil
	}
	if maxOut.Cmp(binReserveOut) <= 0 {
		return maxOut, new(uint256.Int).Set(amountInNormalized), nil
	}

	// Bin depleted: output capped at the bin reserve; back out the consumed input, ceiling.
	binReserveOutNative := new(uint256.Int).Div(binReserveOut, scaleOut)
	var consumedNative *uint256.Int
	if swapForY {
		consumedNative, err = mulDivCeil(binReserveOutNative, priceDenominator, normPrice)
	} else {
		consumedNative, err = mulDivCeil(binReserveOutNative, normPrice, priceDenominator)
	}
	if err != nil {
		return nil, nil, err
	}
	consumed := new(uint256.Int).Mul(consumedNative, scaleIn)
	return new(uint256.Int).Set(binReserveOut), consumed, nil
}

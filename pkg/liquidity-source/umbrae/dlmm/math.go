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
// mirroring the DEPLOYED BinHelper.getPriceFromId exactly: plain 1e18 exponentiation-by-squaring
// (base = numerator*1e18/denominator; result = result*base/1e18; base = base*base/1e18), including
// the saturate-on-overflow guard. (Note: this is NOT the 128.128 algorithm in the stale source.)
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

// getPriceFromId returns the bin price in Y native decimals (Y per 1 whole X), mirroring the
// deployed BinHelper.getPriceFromId.
func getPriceFromId(binID uint32, binStep uint16, decimalsY uint8) (*uint256.Int, error) {
	normalized := getNormalizedPriceFromId(binID, binStep)
	if decimalsY <= 18 {
		return new(uint256.Int).Div(normalized, pow10(18-decimalsY)), nil
	}
	return new(uint256.Int).Mul(normalized, pow10(decimalsY-18)), nil
}

// calculateDynamicFee mirrors the DEPLOYED FeeHelper.calculateDynamicFee: baseFee + quadratic
// variable fee (capped at variableFeeCap when nonzero), total capped at MAX_FEE. Returns the fee
// rate in basis points.
func calculateDynamicFee(baseFactor, variableFeeControl uint16, volatility *uint256.Int, binStep, variableFeeCap uint16) *uint256.Int {
	totalFee := uint256.NewInt(uint64(baseFactor))
	if variableFeeControl != 0 {
		prod := new(uint256.Int).Mul(volatility, uint256.NewInt(uint64(binStep)))
		prod.Mul(prod, prod)
		prod.Mul(prod, uint256.NewInt(uint64(variableFeeControl)))
		prod.Div(prod, e10)
		if variableFeeCap > 0 {
			cap := uint256.NewInt(uint64(variableFeeCap))
			if prod.Cmp(cap) > 0 {
				prod = cap
			}
		}
		totalFee.Add(totalFee, prod)
	}
	if totalFee.Cmp(uMaxFee) > 0 {
		return new(uint256.Int).Set(uMaxFee)
	}
	return totalFee
}

// applyVolatilityDecay mirrors FeeHelper.applyVolatilityDecay: linear decay over decayPeriod, with
// a floor of 1 to avoid premature rounding to 0. Used by the tracker to decay the accumulator to
// the tracked block.
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

// getFeeAmountFrom mirrors FeeHelper.getFeeAmountFrom: amount * totalFee / (10000 + totalFee).
func getFeeAmountFrom(amount, totalFee *uint256.Int) *uint256.Int {
	var num, den uint256.Int
	num.Mul(amount, totalFee)
	den.Add(uBP, totalFee)
	return num.Div(&num, &den)
}

// simulateBinSwap mirrors UmbraeLBPairUpgradeable._simulateBinSwap. All amounts are in the
// 18-decimal normalized space. binReserveOut is the output-side reserve of the bin (reserveY when
// swapForY, else reserveX).
func simulateBinSwap(
	binReserveOut, amountInNormalized, price, precisionX, scaleIn, scaleOut *uint256.Int,
	swapForY bool,
) (amountOutNormalized, amountInLeftNormalized *uint256.Int) {
	amountInNative := new(uint256.Int).Div(amountInNormalized, scaleIn)

	maxAmountOutNative := new(uint256.Int)
	if swapForY {
		maxAmountOutNative.Mul(amountInNative, price)
		maxAmountOutNative.Div(maxAmountOutNative, precisionX)
	} else {
		maxAmountOutNative.Mul(amountInNative, precisionX)
		maxAmountOutNative.Div(maxAmountOutNative, price)
	}
	maxAmountOut := new(uint256.Int).Mul(maxAmountOutNative, scaleOut)

	if maxAmountOut.Cmp(binReserveOut) <= 0 {
		return maxAmountOut, uint256.NewInt(0)
	}

	// Bin depleted: output capped at the bin reserve; back out the consumed input.
	amountOutNormalized = new(uint256.Int).Set(binReserveOut)
	binReserveOutNative := new(uint256.Int).Div(binReserveOut, scaleOut)
	consumedNative := new(uint256.Int)
	if swapForY {
		consumedNative.Mul(binReserveOutNative, precisionX)
		consumedNative.Div(consumedNative, price)
	} else {
		consumedNative.Mul(binReserveOutNative, price)
		consumedNative.Div(consumedNative, precisionX)
	}
	consumed := new(uint256.Int).Mul(consumedNative, scaleIn)
	amountInLeftNormalized = new(uint256.Int)
	if amountInNormalized.Cmp(consumed) > 0 {
		amountInLeftNormalized.Sub(amountInNormalized, consumed)
	}
	return amountOutNormalized, amountInLeftNormalized
}

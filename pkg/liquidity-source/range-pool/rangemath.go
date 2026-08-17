package rangepool

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/math"
)

// AbsoluteMinTokenBalance is RangeMath.ABSOLUTE_MIN_TOKEN_BALANCE (scaled18): the
// pool must always retain at least this much of every token, so an exact-in swap's
// output is capped at factBalanceOut - AbsoluteMinTokenBalance.
var AbsoluteMinTokenBalance = uint256.NewInt(1e6)

// CalcOutGivenIn is a bit-exact port of RangeMath.calcOutGivenIn: the weighted-product
// curve on VIRTUAL balances, with the result capped by the fact (real) output balance.
//
//	aO = bO * (1 - (bI / (bI + aI))^(wI/wO));  then  aO = min(aO, factBalanceOut - 1e6)
//
// Unlike balancer/v3/math.WeightedMath.ComputeOutGivenExactIn, RangeMath has NO 30%
// MAX_IN_RATIO guard — the fact-balance cap is the only limiter (reference §3). The two
// agree bit-for-bit for sub-30% amounts (asserted in rangemath_test.go); this function
// additionally returns the capped value for larger inputs instead of erroring.
func CalcOutGivenIn(
	virtualBalanceIn,
	weightIn,
	virtualBalanceOut,
	weightOut,
	amountIn,
	factBalanceOut *uint256.Int,
) (*uint256.Int, error) {
	denominator, err := math.FixPoint.Add(virtualBalanceIn, amountIn)
	if err != nil {
		return nil, err
	}

	base, err := math.FixPoint.DivUp(virtualBalanceIn, denominator)
	if err != nil {
		return nil, err
	}

	exponent, err := math.FixPoint.DivDown(weightIn, weightOut)
	if err != nil {
		return nil, err
	}

	power, err := math.FixPoint.PowUp(base, exponent)
	if err != nil {
		return nil, err
	}

	uncapped, err := math.FixPoint.MulDown(virtualBalanceOut, math.FixPoint.Complement(power))
	if err != nil {
		return nil, err
	}

	// maxOut = factBalanceOut > 1e6 ? factBalanceOut - 1e6 : 0
	maxOut := new(uint256.Int)
	if factBalanceOut.Gt(AbsoluteMinTokenBalance) {
		maxOut.Sub(factBalanceOut, AbsoluteMinTokenBalance)
	}

	if uncapped.Lt(maxOut) {
		return uncapped, nil
	}
	return maxOut, nil
}

// CalcInGivenOut is a bit-exact port of RangeMath.calcInGivenOut: the amountIn required
// for a given amountOut, on VIRTUAL balances.
//
//	aI = bI * ((bO / (bO - aO))^(wO/wI) - 1)
//
// Callers MUST first check IsExactOutAllowed (the onSwap guards) — this function assumes
// amountOut < virtualBalanceOut. Like RangeMath, it has NO 30% MAX_OUT_RATIO guard.
func CalcInGivenOut(
	virtualBalanceIn,
	weightIn,
	virtualBalanceOut,
	weightOut,
	amountOut *uint256.Int,
) (*uint256.Int, error) {
	// Short-circuit: zero out requires zero in (matches the Solidity guard that avoids a
	// dust charge from powUp rounding on base == 1e18).
	if amountOut.IsZero() {
		return new(uint256.Int), nil
	}

	delta, err := math.FixPoint.Sub(virtualBalanceOut, amountOut)
	if err != nil {
		return nil, err
	}

	base, err := math.FixPoint.DivUp(virtualBalanceOut, delta)
	if err != nil {
		return nil, err
	}

	exponent, err := math.FixPoint.DivUp(weightOut, weightIn)
	if err != nil {
		return nil, err
	}

	power, err := math.FixPoint.PowUp(base, exponent)
	if err != nil {
		return nil, err
	}

	ratio, err := math.FixPoint.Sub(power, math.U1e18)
	if err != nil {
		return nil, err
	}

	return math.FixPoint.MulUp(virtualBalanceIn, ratio)
}

// IsExactOutAllowed mirrors the RangePool.onSwap guards for EXACT_OUT. It returns false
// (treat as no liquidity — the Vault would revert) if either:
//   - amountOut + AbsoluteMinTokenBalance > factBalanceOut (would drain the pool), or
//   - amountOut >= virtualBalanceOut (curve base non-positive / undefined).
func IsExactOutAllowed(amountOut, factBalanceOut, virtualBalanceOut *uint256.Int) bool {
	sum, overflow := new(uint256.Int).AddOverflow(amountOut, AbsoluteMinTokenBalance)
	if overflow || sum.Gt(factBalanceOut) {
		return false
	}
	if amountOut.Cmp(virtualBalanceOut) >= 0 {
		return false
	}
	return true
}

// CalcSpotPrice is a bit-exact port of RangeMath.calcSpotPrice: the spot price of the
// quote token in base-token terms (scaled18), base per 1 quote.
//
//	price = (Vb / Vq) * Wq / Wb   (as divDown -> mulDown -> divDown)
func CalcSpotPrice(
	virtualBalanceBase,
	weightBase,
	virtualBalanceQuote,
	weightQuote *uint256.Int,
) (*uint256.Int, error) {
	ratio, err := math.FixPoint.DivDown(virtualBalanceBase, virtualBalanceQuote)
	if err != nil {
		return nil, err
	}

	scaled, err := math.FixPoint.MulDown(ratio, weightQuote)
	if err != nil {
		return nil, err
	}

	return math.FixPoint.DivDown(scaled, weightBase)
}

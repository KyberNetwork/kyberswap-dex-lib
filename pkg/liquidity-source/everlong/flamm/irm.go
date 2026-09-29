package everlongflamm

import (
	"math/big"

	"github.com/holiman/uint256"
)

// The market's rate model: Morpho's AdaptiveCurveIrm (0x46415998764C29aB2a25CbeA6254146D50D22687 on Base, the
// IRM of the c104 venue market 0x9103c3b4…1836 per Blue's idToMarketParams), ported from
// morpho-org/morpho-blue-irm v1.0.0 (AdaptiveCurveIrm.sol, ExpLib.sol, ConstantsLib.sol, MathLib.sol). The
// arithmetic is int256: wMulToZero / wDivToZero truncate toward zero (big.Int.Quo), wExp decomposes over ln 2
// and shifts a positive e^r. Given uint128 market totals no checked int256 operation in _borrowRate can overflow
// (the largest, `coeff*err`, stays below 2^253), so the only revert is `block.timestamp - lastUpdate`
// underflowing. Validated against the deployed IRM over rateAtTarget x utilisation x elapsed (mm_irm_grid.json.gz).

var (
	mmIrmWad               = big.NewInt(1e18)
	mmIrmCurveSteepness    = big.NewInt(4e18)
	mmIrmAdjustmentSpeed   = big.NewInt(50_000_000_000_000_000_000 / (365 * 86400))
	mmIrmTargetUtilization = big.NewInt(9e17)
	mmIrmInitialRate       = big.NewInt(40_000_000_000_000_000 / (365 * 86400))
	mmIrmMinRate           = big.NewInt(1_000_000_000_000_000 / (365 * 86400))
	mmIrmMaxRate           = big.NewInt(2_000_000_000_000_000_000 / (365 * 86400))
	mmIrmLn2               = big.NewInt(693147180559945309)
	mmIrmLnWei, _          = new(big.Int).SetString("-41446531673892822312", 10)
	mmIrmExpUpperBound, _  = new(big.Int).SetString("93859467695000404319", 10)
	mmIrmExpUpperValue, _  = new(big.Int).SetString("57716089161558943949701069502944508345128422502756744429568", 10)
	mmIrmErrNormAbove      = new(big.Int).Sub(mmIrmWad, mmIrmTargetUtilization) // WAD - TARGET_UTILIZATION
	mmIrmCoeffBelow        = new(big.Int).Sub(mmIrmWad, new(big.Int).Quo(new(big.Int).Mul(mmIrmWad, mmIrmWad), mmIrmCurveSteepness))
	mmIrmCoeffAbove        = new(big.Int).Sub(mmIrmCurveSteepness, mmIrmWad)
	mmIrmHalfLn2           = new(big.Int).Quo(mmIrmLn2, big.NewInt(2))
	mmIrmTwo               = big.NewInt(2)
	mmIrmFour              = big.NewInt(4)
)

// mmIrmBorrowRate is AdaptiveCurveIrm._borrowRate(id, market) at `now`: (avgRate, endRateAtTarget). The view
// (borrowRateView) returns avgRate; Blue's accrual calls borrowRate, which also stores endRateAtTarget.
func mmIrmBorrowRate(m *mmMarket, rateAtTarget *uint256.Int, now uint64) (uint256.Int, uint256.Int, error) {
	var avgOut, endOut uint256.Int
	// utilization = tsa > 0 ? wDivDown(tba, tsa) : 0 -- MathLib.wDivDown on the uint128 totals
	utilization := new(big.Int)
	if !m.TotalSupplyAssets.IsZero() {
		u, _ := mmWDivDown(&m.TotalBorrowAssets, &m.TotalSupplyAssets) // tba*WAD < 2^188: never overflows
		utilization = u.ToBig()
	}
	errNormFactor := mmIrmTargetUtilization
	if utilization.Cmp(mmIrmTargetUtilization) > 0 {
		errNormFactor = mmIrmErrNormAbove
	}
	e := new(big.Int).Sub(utilization, mmIrmTargetUtilization)
	e.Mul(e, mmIrmWad).Quo(e, errNormFactor) // wDivToZero

	start := rateAtTarget.ToBig()
	var avgRateAtTarget, endRateAtTarget *big.Int
	if start.Sign() == 0 {
		// first interaction
		avgRateAtTarget, endRateAtTarget = mmIrmInitialRate, mmIrmInitialRate
	} else {
		if now < m.LastUpdate.Uint64() || !m.LastUpdate.IsUint64() {
			return avgOut, endOut, errPanicArithmetic
		}
		speed := new(big.Int).Mul(mmIrmAdjustmentSpeed, e)
		speed.Quo(speed, mmIrmWad) // wMulToZero
		elapsed := new(big.Int).SetUint64(now - m.LastUpdate.Uint64())
		linearAdaptation := new(big.Int).Mul(speed, elapsed)
		if linearAdaptation.Sign() == 0 {
			avgRateAtTarget, endRateAtTarget = start, start
		} else {
			// trapezoid with N = 2: (start + end + 2*mid) / 4, each leg bounded to [MIN, MAX]
			endRateAtTarget = mmIrmNewRateAtTarget(start, linearAdaptation)
			half := new(big.Int).Quo(linearAdaptation, mmIrmTwo)
			mid := mmIrmNewRateAtTarget(start, half)
			avgRateAtTarget = new(big.Int).Add(start, endRateAtTarget)
			avgRateAtTarget.Add(avgRateAtTarget, new(big.Int).Mul(mmIrmTwo, mid))
			avgRateAtTarget.Quo(avgRateAtTarget, mmIrmFour)
		}
	}
	// Blue's timestamp subtraction runs only on the adapting branch; the first interaction never reads it.
	avgOut.SetFromBig(mmIrmCurve(avgRateAtTarget, e))
	endOut.SetFromBig(endRateAtTarget)
	return avgOut, endOut, nil
}

// mmIrmCurve is AdaptiveCurveIrm._curve: ((1-1/C)*err + 1) * rate below target, ((C-1)*err + 1) * rate above,
// both products wMulToZero.
func mmIrmCurve(rate, e *big.Int) *big.Int {
	coeff := mmIrmCoeffAbove
	if e.Sign() < 0 {
		coeff = mmIrmCoeffBelow
	}
	z := new(big.Int).Mul(coeff, e)
	z.Quo(z, mmIrmWad).Add(z, mmIrmWad)
	z.Mul(z, rate)
	return z.Quo(z, mmIrmWad)
}

// mmIrmNewRateAtTarget is AdaptiveCurveIrm._newRateAtTarget: wMulToZero(start, wExp(linearAdaptation)) bounded
// to [MIN_RATE_AT_TARGET, MAX_RATE_AT_TARGET].
func mmIrmNewRateAtTarget(start, linearAdaptation *big.Int) *big.Int {
	z := new(big.Int).Mul(start, mmIrmWExp(linearAdaptation))
	z.Quo(z, mmIrmWad)
	if z.Cmp(mmIrmMaxRate) > 0 {
		z.Set(mmIrmMaxRate)
	}
	if z.Cmp(mmIrmMinRate) < 0 {
		z.Set(mmIrmMinRate)
	}
	return z
}

// mmIrmWExp is ExpLib.wExp: zero below ln(1e-18), clipped at WEXP_UPPER_BOUND, else x = q*ln2 + r with q rounded
// half toward zero, e^r by a 2nd-order Taylor polynomial `WAD + r + r*r/WAD/2`, and e^x = e^r << q (>> -q).
func mmIrmWExp(x *big.Int) *big.Int {
	if x.Cmp(mmIrmLnWei) < 0 {
		return new(big.Int)
	}
	if x.Cmp(mmIrmExpUpperBound) >= 0 {
		return new(big.Int).Set(mmIrmExpUpperValue)
	}
	adj := mmIrmHalfLn2
	if x.Sign() < 0 {
		adj = new(big.Int).Neg(mmIrmHalfLn2)
	}
	q := new(big.Int).Add(x, adj)
	q.Quo(q, mmIrmLn2)
	r := new(big.Int).Mul(q, mmIrmLn2)
	r.Sub(x, r)
	rr := new(big.Int).Mul(r, r)
	rr.Quo(rr, mmIrmWad).Quo(rr, mmIrmTwo)
	expR := new(big.Int).Add(mmIrmWad, r)
	expR.Add(expR, rr)
	// expR > 0 (|r| <= ln2/2), so the arithmetic shift right is a plain floor shift
	if q.Sign() >= 0 {
		return expR.Lsh(expR, uint(q.Uint64()))
	}
	return expR.Rsh(expR, uint(new(big.Int).Neg(q).Uint64()))
}

package fermi

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Pricing flow (re-implements FermiEngine on-chain math against CurveData):
//
//	inv_ratio = (vaultIn - safety) * 1e18 / (vaultIn + vaultOut)
//	inv_adj   = evalSpline(inventorySpline, inv_ratio)   // bps × 1e18
//	size_norm = amountIn * 1e18 / scalingDenominator
//	size_adj  = evalSpline(sizeSpline, size_norm)        // bps × 1e18
//	eff_price = priceFactor(midPrice + size_adj, inv_adj)
//	amountOut = ConvertOut(amountIn, eff_price, decScales)
//
// All scalar coefficients are signed int128; arithmetic is signed throughout.

var (
	ErrCurveNotAvailable  = errors.New("curve data not available on this pool")
	ErrKnotOutOfRange     = errors.New("input out of spline range")
	ErrEmptySpline        = errors.New("spline has no knots")
	ErrZeroEffectivePrice = errors.New("zero effective price")
	ErrInvalidCurveData   = errors.New("malformed curve data")
)

var (
	oneE18   = bignumber.BONE
	oneE8    = bignumber.TenPowInt(8)
	bpsScale = bignumber.BasisPoint
	bpsTimes = new(big.Int).Mul(bpsScale, oneE18)
)

// evalCubic returns y = c0*1e18 + c1*t + c2*(t²/1e18) + c3*(t³/1e18).
// Mirrors FermiEngine private function 0x3dcb; t ∈ [0, 1e18].
func evalCubic(t, c0, c1, c2, c3 *big.Int) *big.Int {
	tSq := new(big.Int).Mul(t, t)
	tSq.Quo(tSq, oneE18)
	tCu := new(big.Int).Mul(tSq, t)
	tCu.Quo(tCu, oneE18)

	v4 := new(big.Int).Mul(c0, oneE18)
	v5 := new(big.Int).Mul(c1, t)
	v6 := new(big.Int).Mul(c2, tSq)
	v7 := new(big.Int).Mul(c3, tCu)

	out := new(big.Int).Add(v7, v6)
	out.Add(out, v5)
	out.Add(out, v4)
	return out
}

// parseKnot decodes JSON-string fields of a Knot into signed big.Ints.
func parseKnot(k *Knot) (xLo, xHi, c0, c1, c2, c3 *big.Int, ok bool) {
	xLo = bignumber.NewBig(k.XLo)
	xHi = bignumber.NewBig(k.XHi)
	c0 = bignumber.NewBig(k.C0)
	c1 = bignumber.NewBig(k.C1)
	c2 = bignumber.NewBig(k.C2)
	c3 = bignumber.NewBig(k.C3)
	ok = xLo != nil && xHi != nil && c0 != nil && c1 != nil && c2 != nil && c3 != nil
	return
}

// evalSpline finds the knot whose [xLo, xHi] bracket contains x and returns
// the cubic evaluated at t = (x - xLo) * 1e18 / (xHi - xLo).
// Mirrors engine loop 0x29ba (binary search over sorted knots).
func evalSpline(knots []Knot, x *big.Int) (*big.Int, error) {
	if len(knots) == 0 {
		return nil, ErrEmptySpline
	}

	lo, n := 0, len(knots)
	for lo < n {
		mid := (lo + n) / 2
		_, xHi, _, _, _, _, ok := parseKnot(&knots[mid])
		if !ok {
			return nil, ErrInvalidCurveData
		}
		if x.Cmp(xHi) > 0 {
			lo = mid + 1
		} else {
			n = mid
		}
	}
	if lo >= len(knots) {
		return nil, ErrKnotOutOfRange
	}

	xLo, xHi, c0, c1, c2, c3, ok := parseKnot(&knots[lo])
	if !ok {
		return nil, ErrInvalidCurveData
	}
	if x.Cmp(xLo) < 0 {
		return nil, ErrKnotOutOfRange
	}

	span := new(big.Int).Sub(xHi, xLo)
	if span.Sign() <= 0 {
		return nil, ErrInvalidCurveData
	}
	dx := new(big.Int).Sub(x, xLo)
	t := new(big.Int).Mul(dx, oneE18)
	t.Quo(t, span)

	return evalCubic(t, c0, c1, c2, c3), nil
}

// priceFactor mirrors on-chain helper 0x39d3:
//
//	out = price * (10000*1e18) / (10000*1e18 + adj)
//
// Returns nil when the denominator collapses to zero.
func priceFactor(price, adj *big.Int) *big.Int {
	denom := new(big.Int).Add(bpsTimes, adj)
	if denom.Sign() <= 0 {
		return nil
	}
	num := new(big.Int).Mul(price, bpsTimes)
	return num.Quo(num, denom)
}

// inventoryRatio computes the normalized vault-inventory parameter for the
// inventory spline. Replicates lines 299-306 of source.txt (function 0x29ba):
//
//	totalIn  = vaultBal0 * midPrice / 1e8 * ds1 / ds0  // token0 → token1 units
//	totalAll = totalIn + vaultBal1
//	ratio    = (totalIn - totalAll*safetyFeeBps/10000) * 1e18 / totalAll
//
// ds0/ds1 are the CANONICAL decimal scales (not direction-dependent).
func inventoryRatio(
	vaultIn, vaultOut *big.Int,
	midPrice, tokenInDecScale, tokenOutDecScale *big.Int,
	safetyFeeBps uint16,
) (*big.Int, error) {
	totalIn := new(big.Int).Mul(vaultIn, midPrice)
	totalIn.Quo(totalIn, oneE8)
	totalIn.Mul(totalIn, tokenOutDecScale)
	totalIn.Quo(totalIn, tokenInDecScale)

	totalAll := new(big.Int).Add(totalIn, vaultOut)
	if totalAll.Sign() <= 0 {
		return nil, ErrInvalidCurveData
	}

	fee := new(big.Int).SetInt64(int64(safetyFeeBps))
	fee.Mul(totalAll, fee)
	fee.Quo(fee, bpsScale)

	totalIn.Sub(totalIn, fee)    // netIn = totalIn - fee
	totalIn.Mul(totalIn, oneE18) // ratio = netIn * 1e18 / totalAll
	totalIn.Quo(totalIn, totalAll)
	return totalIn, nil
}

// convertOutForward applies the effective price for the canonical direction
// (engine helper 0x36cb):
//
//	amountOut = amountIn * dsOut * effPrice / (1e8 * dsIn)
func convertOutForward(amountIn, effPrice, ds0, ds1 *big.Int) *big.Int {
	num := new(big.Int).Mul(amountIn, ds1)
	num.Mul(num, effPrice)
	denom := new(big.Int).Mul(oneE8, ds0)
	if denom.Sign() <= 0 {
		return new(big.Int)
	}
	return num.Quo(num, denom)
}

// convertOutReverse applies the effective price for the reverse direction
// (engine helper 0x371a):
//
//	amountOut = amountIn * ds0 * 1e8 / (effPrice * ds1)
func convertOutReverse(amountIn, effPrice, ds0, ds1 *big.Int) *big.Int {
	num := new(big.Int).Mul(amountIn, ds0)
	num.Mul(num, oneE8)
	denom := new(big.Int).Mul(effPrice, ds1)
	if denom.Sign() <= 0 {
		return new(big.Int)
	}
	return num.Quo(num, denom)
}

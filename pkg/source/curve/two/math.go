package two

import (
	"github.com/holiman/uint256"
)

// Math for the legacy Curve v2 two-coin pools (CurveCryptoSwap2), ported from the previous math/big implementation.
// Every expression keeps the original operation order, so results are bit-identical to the math/big version for
// all inputs where no intermediate value exceeds 2**256 or goes below 0 (the contract reverts in those cases; here
// they are reported as an error instead of producing a value). Prices are assumed to be below 2**128, as the
// contract stores them.

// arith records the first overflow / underflow / division by zero so formulas can be written in the same
// shape as the Vyper source and checked once.
type arith struct{ err error }

func (a *arith) fail(err error) {
	if a.err == nil {
		a.err = err
	}
}

func (a *arith) add(x, y uint256.Int) (z uint256.Int) {
	if _, overflow := z.AddOverflow(&x, &y); overflow {
		a.fail(ErrOverflow)
	}
	return z
}

func (a *arith) sub(x, y uint256.Int) (z uint256.Int) {
	if _, underflow := z.SubOverflow(&x, &y); underflow {
		a.fail(ErrUnderflow)
	}
	return z
}

func (a *arith) mul(x, y uint256.Int) (z uint256.Int) {
	if _, overflow := z.MulOverflow(&x, &y); overflow {
		a.fail(ErrOverflow)
	}
	return z
}

func (a *arith) div(x, y uint256.Int) (z uint256.Int) {
	if y.IsZero() {
		a.fail(ErrDenominatorZero)
		return z
	}
	return *z.Div(&x, &y)
}

func absDiff(x, y uint256.Int) (z uint256.Int) {
	if x.Gt(&y) {
		return *z.Sub(&x, &y)
	}
	return *z.Sub(&y, &x)
}

// sortDesc sorts in descending order.
func sortDesc(x [nCoins]uint256.Int) [nCoins]uint256.Int {
	if x[0].Lt(&x[1]) {
		x[0], x[1] = x[1], x[0]
	}
	return x
}

func geometricMean(x [nCoins]uint256.Int, sort bool) (uint256.Int, error) {
	var a arith
	if sort {
		x = sortDesc(x)
	}
	D := x[0]
	for range 255 {
		Dprev := D
		tmp := bone
		for k := range nCoins {
			tmp = a.div(a.mul(tmp, x[k]), D)
		}
		D = a.div(a.mul(D, a.add(bone, tmp)), twoBone) // D * ((N-1) * 10**18 + tmp) / (N * 10**18)
		if a.err != nil {
			return zero, a.err
		}
		diff := absDiff(D, Dprev)
		if !diff.Gt(&one) {
			return D, nil
		}
		if m := a.mul(diff, bone); a.err == nil && m.Lt(&D) {
			return D, nil
		}
		if a.err != nil {
			return zero, a.err
		}
	}
	return zero, errDidNotConverge
}

func sqrtInt(x uint256.Int) (uint256.Int, error) {
	if x.IsZero() {
		return zero, nil
	}
	var a arith
	z := a.div(a.add(x, bone), two)
	y := x
	for range 256 {
		if z.Eq(&y) {
			return y, nil
		}
		y = z
		z = a.div(a.add(a.div(a.mul(x, bone), z), z), two)
		if a.err != nil {
			return zero, a.err
		}
	}
	return zero, errSqrtDidNotConverge
}

func newtonD(ANN, gamma uint256.Int, xUnsorted [nCoins]uint256.Int) (uint256.Int, error) {
	if gamma.Lt(MinGamma) || gamma.Gt(MaxGamma) {
		return zero, errUnsafeGamma
	}
	var a arith
	x := sortDesc(xUnsorted)
	if x[0].Lt(&tenPow9) || x[0].Gt(&tenPow33) {
		return zero, errUnsafeX0
	}
	for i := 1; i < nCoins; i++ {
		frac := a.div(a.mul(x[i], bone), x[0])
		if a.err != nil {
			return zero, a.err
		}
		if frac.Lt(&tenPow11) {
			return zero, errUnsafeX
		}
	}
	mean, err := geometricMean(x, false)
	if err != nil {
		return zero, err
	}
	D := a.mul(nCoinsU, mean)
	var S uint256.Int
	for k := range nCoins {
		S = a.add(S, x[k])
	}
	if a.err != nil {
		return zero, a.err
	}
	g1 := a.add(gamma, bone) // loop invariant: gamma + 10**18
	for range 255 {
		Dprev := D
		K0 := bone
		for k := range nCoins {
			K0 = a.div(a.mul(a.mul(K0, x[k]), nCoinsU), D)
		}
		var g1k0 uint256.Int
		if g1.Gt(&K0) {
			g1k0 = a.add(a.sub(g1, K0), one)
		} else {
			g1k0 = a.add(a.sub(K0, g1), one)
		}
		mul1 := a.div(a.mul(a.mul(a.div(a.mul(a.div(a.mul(bone, D), gamma), g1k0), gamma), g1k0), *AMultiplier), ANN)
		mul2 := a.div(a.mul(fourBone, K0), g1k0)
		negFprime := a.sub(
			a.add(a.add(S, a.div(a.mul(S, mul2), bone)), a.div(a.mul(mul1, nCoinsU), K0)),
			a.div(a.mul(mul2, D), bone),
		)
		Dplus := a.div(a.mul(D, a.add(negFprime, S)), negFprime)
		Dminus := a.div(a.mul(D, D), negFprime)
		if bone.Gt(&K0) {
			Dminus = a.add(Dminus, a.div(a.mul(a.div(a.mul(D, a.div(mul1, negFprime)), bone), a.sub(bone, K0)), K0))
		} else {
			Dminus = a.sub(Dminus, a.div(a.mul(a.div(a.mul(D, a.div(mul1, negFprime)), bone), a.sub(K0, bone)), K0))
		}
		if a.err != nil {
			return zero, a.err
		}
		if Dplus.Gt(&Dminus) {
			D = a.sub(Dplus, Dminus)
		} else {
			D = a.div(a.sub(Dminus, Dplus), two)
		}
		diff := absDiff(D, Dprev)
		temp := tenPow16
		if D.Gt(&temp) {
			temp = D
		}
		if m := a.mul(diff, tenPow14); a.err == nil && m.Lt(&temp) {
			for k := range nCoins {
				frac := a.div(a.mul(x[k], bone), D)
				if a.err != nil {
					return zero, a.err
				}
				if frac.Lt(&tenPow16) || frac.Gt(&tenPow20) {
					return zero, errUnsafeX
				}
			}
			return D, nil
		}
		if a.err != nil {
			return zero, a.err
		}
	}
	return zero, errDidNotConverge
}

// newtonY calculates x[i] given the other balance and D.
// Reference: https://github.com/curvefi/curve-crypto-contract/blob/master/contracts/two/CurveCryptoSwap2ETH.vy#L346-L349
func newtonY(ann, gamma uint256.Int, x [nCoins]uint256.Int, D uint256.Int, i int) (uint256.Int, error) {
	if ann.Lt(MinA) || ann.Gt(MaxA) {
		return zero, errUnsafeA
	}
	if gamma.Lt(MinGamma) || gamma.Gt(MaxGamma) {
		return zero, errUnsafeGamma
	}
	if D.Lt(&tenPow17) || D.Gt(&tenPow33) {
		return zero, errUnsafeD
	}
	var a arith
	y := a.div(D, nCoinsU)
	K0i := bone
	var Si uint256.Int

	xs := x
	xs[i] = zero
	xs = sortDesc(xs)
	convergenceLimit := a.div(xs[0], tenPow14)
	if temp := a.div(D, tenPow14); temp.Gt(&convergenceLimit) {
		convergenceLimit = temp
	}
	if hundred.Gt(&convergenceLimit) {
		convergenceLimit = hundred
	}

	for j := 2; j < nCoins+1; j++ {
		_x := xs[nCoins-j]
		if _x.IsZero() {
			return zero, ErrDenominatorZero
		}
		y = a.div(a.mul(y, D), a.mul(_x, nCoinsU))
		Si = a.add(Si, _x)
	}
	for j := 0; j < nCoins-1; j++ {
		K0i = a.div(a.mul(a.mul(K0i, xs[j]), nCoinsU), D)
	}

	// Loop invariants of the original per-iteration expressions.
	bd := a.mul(bone, D)     // 10**18 * D
	bdg := a.div(bd, gamma)  // 10**18 * D / gamma
	g1 := a.add(gamma, bone) // gamma + 10**18
	if a.err != nil {
		return zero, a.err
	}

	for range 255 {
		yPrev := y
		K0 := a.div(a.mul(a.mul(K0i, y), nCoinsU), D)
		S := a.add(Si, y)
		var g1k0 uint256.Int
		if g1.Gt(&K0) {
			g1k0 = a.add(a.sub(g1, K0), one)
		} else {
			g1k0 = a.add(a.sub(K0, g1), one)
		}
		mul1 := a.div(a.mul(a.div(a.mul(bdg, g1k0), gamma), a.mul(g1k0, *AMultiplier)), ann)
		mul2 := a.add(a.div(a.mul(twoBone, K0), g1k0), bone)
		yfprime := a.add(a.add(a.mul(bone, y), a.mul(S, mul2)), mul1)
		dyfprime := a.mul(D, mul2)
		if a.err != nil {
			return zero, a.err
		}
		if yfprime.Lt(&dyfprime) {
			y = a.div(yPrev, two)
			continue
		}
		yfprime = a.sub(yfprime, dyfprime)

		if y.IsZero() {
			return zero, ErrDenominatorZero
		}
		fprime := a.div(yfprime, y)
		if fprime.IsZero() {
			return zero, ErrDenominatorZero
		}

		yMinus := a.div(mul1, fprime)
		yPlus := a.add(a.div(a.add(yfprime, bd), fprime), a.div(a.mul(yMinus, bone), K0))
		yMinus = a.add(yMinus, a.div(a.mul(bone, S), fprime))
		if a.err != nil {
			return zero, a.err
		}
		if yPlus.Lt(&yMinus) {
			y = a.div(yPrev, two)
		} else {
			y = a.sub(yPlus, yMinus)
		}
		diff := absDiff(y, yPrev)
		t := a.div(y, tenPow14)
		if convergenceLimit.Gt(&t) {
			t = convergenceLimit
		}
		if diff.Lt(&t) {
			frac := a.div(a.mul(y, bone), D)
			if a.err != nil {
				return zero, a.err
			}
			if frac.Lt(&tenPow16) || frac.Gt(&tenPow20) {
				return zero, errUnsafeY
			}
			return y, nil
		}
	}
	return zero, errDidNotConverge
}

func reductionCoefficient(x [nCoins]uint256.Int, feeGamma uint256.Int) (uint256.Int, error) {
	var a arith
	K := bone
	var S uint256.Int
	for k := range nCoins {
		S = a.add(S, x[k])
	}
	for k := range nCoins {
		K = a.div(a.mul(a.mul(K, nCoinsU), x[k]), S)
	}
	if !feeGamma.IsZero() {
		K = a.div(a.mul(feeGamma, bone), a.sub(a.add(feeGamma, bone), K))
	}
	return K, a.err
}

func halfpow(power, precision uint256.Int) (uint256.Int, error) {
	var a arith
	intpow := a.div(power, bone)
	otherpow := a.sub(power, a.mul(intpow, bone))
	if intpow.GtUint64(59) {
		return zero, nil
	}
	var result uint256.Int
	result.Rsh(&bone, uint(intpow.Uint64())) // 10**18 / 2**intpow
	if otherpow.IsZero() {
		return result, nil
	}
	term := bone
	S := bone
	neg := false
	for i := uint64(1); i < 256; i++ {
		var K uint256.Int
		K.SetUint64(i)
		K = a.mul(K, bone)
		c := a.sub(K, bone)
		if otherpow.Gt(&c) {
			c = a.sub(otherpow, c)
			neg = !neg
		} else {
			c = a.sub(c, otherpow)
		}
		term = a.div(a.mul(term, a.div(a.mul(c, halfBone), bone)), K)
		if neg {
			S = a.sub(S, term)
		} else {
			S = a.add(S, term)
		}
		if a.err != nil {
			return zero, a.err
		}
		if term.Lt(&precision) {
			result = a.div(a.mul(result, S), bone)
			return result, a.err
		}
	}
	return zero, errDidNotConverge
}

// aGamma returns [A, gamma], interpolated while a ramp is in progress.
func (t *PoolSimulator) aGamma(now int64) ([2]uint256.Int, error) {
	var A1, gamma1 uint256.Int
	gamma1.And(&t.FutureAGamma, PriceMask)
	A1.Rsh(&t.FutureAGamma, 128)
	if t1 := t.FutureAGammaTime; now < t1 {
		t0 := t.InitialAGammaTime
		// t0 = block.timestamp - t0 underflows (the contract reverts) if the ramp has not started yet. Otherwise
		// t0 <= now < t1, so every time difference below is positive.
		if now < t0 {
			return [2]uint256.Int{}, ErrUnderflow
		}
		var dt1, dt0, dt2 uint256.Int
		dt1.SetUint64(uint64(t1 - t0))  // t1 -= t0
		dt0.SetUint64(uint64(now - t0)) // t0 = block.timestamp - t0
		dt2.SetUint64(uint64(t1 - now)) // t2 = t1 - t0 = (t1 - t0) - (block.timestamp - t0)
		var A0, gamma0 uint256.Int
		A0.Rsh(&t.InitialAGamma, 128)
		gamma0.And(&t.InitialAGamma, PriceMask)
		var a arith
		A1 = a.div(a.add(a.mul(A0, dt2), a.mul(A1, dt0)), dt1)
		gamma1 = a.div(a.add(a.mul(gamma0, dt2), a.mul(gamma1, dt0)), dt1)
		if a.err != nil {
			return [2]uint256.Int{}, a.err
		}
	}
	return [2]uint256.Int{A1, gamma1}, nil
}

// feeCalc returns the swap fee rate (10**10 = 100%) for balances xp.
func (t *PoolSimulator) feeCalc(xp [nCoins]uint256.Int) (uint256.Int, error) {
	f, err := reductionCoefficient(xp, t.FeeGamma)
	if err != nil {
		return zero, err
	}
	var a arith
	ret := a.div(a.add(a.mul(t.MidFee, f), a.mul(t.OutFee, a.sub(bone, f))), bone)
	return ret, a.err
}

// balances returns the pool balances (Info.Reserves) as uint256.
func (t *PoolSimulator) balances() (b [nCoins]uint256.Int, err error) {
	for k := range nCoins {
		if b[k], err = toU256(t.Info.Reserves[k]); err != nil {
			return b, err
		}
	}
	return b, nil
}

// getDy is CurveCryptoSwap2.get_dy: the amount out (after fee) for dx of coin i swapped to coin j, and the fee rate
// (not the fee amount, as the previous implementation returned it).
// https://github.com/curvefi/curve-crypto-contract/blob/d7d04cd9ae038970e40be850df99de8c1ff7241b/contracts/two/CurveCryptoSwap2.vy#L842
func (t *PoolSimulator) getDy(i, j int, dx uint256.Int, now int64) (dy, fee uint256.Int, err error) {
	if i == j {
		return zero, zero, errSameToken
	}
	if i < 0 || i >= nCoins || j < 0 || j >= nCoins {
		return zero, zero, errTokenIndexOutOfRange
	}
	var a arith
	priceScale := a.mul(t.PriceScale, t.Precisions[1])

	xp, err := t.balances() // xp: uint256[N_COINS] = self.balances
	if err != nil {
		return zero, zero, err
	}
	xp[i] = a.add(xp[i], dx)
	xp[0] = a.mul(xp[0], t.Precisions[0])
	xp[1] = a.div(a.mul(xp[1], priceScale), *Precision)
	if a.err != nil {
		return zero, zero, a.err
	}

	AGamma, err := t.aGamma(now)
	if err != nil {
		return zero, zero, err
	}
	y, err := newtonY(AGamma[0], AGamma[1], xp, t.D, j)
	if err != nil {
		return zero, zero, err
	}

	dy = a.sub(a.sub(xp[j], y), one) // dy: uint256 = xp[j] - y - 1
	xp[j] = y
	if j > 0 {
		dy = a.div(a.mul(dy, *Precision), priceScale) // dy = dy * PRECISION / price_scale
	} else {
		dy = a.div(dy, t.Precisions[0]) // dy /= precisions[0]
	}
	if a.err != nil {
		return zero, zero, a.err
	}
	// dy -= self._fee(xp) * dy / 10**10
	fee, err = t.feeCalc(xp)
	if err != nil {
		return zero, zero, err
	}
	dy = a.sub(dy, a.div(a.mul(fee, dy), tenPow10))
	return dy, fee, a.err
}

// exchange is CurveCryptoSwap2.exchange: it updates the pool state (balances, D, price oracle, price scale, ...)
// as the contract would and returns dy. now is the block timestamp.
func (t *PoolSimulator) exchange(i, j int, dx uint256.Int, now int64) (uint256.Int, error) {
	if i == j {
		return zero, errSameCoin
	}
	if i >= nCoins || j >= nCoins || i < 0 || j < 0 {
		return zero, errCoinIndexOutOfRange
	}
	if dx.IsZero() {
		return zero, errZeroAmount
	}

	AGamma, err := t.aGamma(now)
	if err != nil {
		return zero, err
	}
	xp, err := t.balances()
	if err != nil {
		return zero, err
	}
	var a arith
	ix := j
	var p uint256.Int

	y := xp[j]
	x0 := xp[i]
	xp[i] = a.add(x0, dx)
	if a.err != nil {
		return zero, a.err
	}
	t.Info.Reserves[i] = xp[i].ToBig()
	priceScale := t.PriceScale

	xp[0] = a.mul(xp[0], t.Precisions[0])
	xp[1] = a.div(a.mul(a.mul(xp[1], priceScale), t.Precisions[1]), *Precision)
	if a.err != nil {
		return zero, a.err
	}

	if ti := t.FutureAGammaTime; ti > 0 {
		x0 = a.mul(x0, t.Precisions[i])
		if i > 0 {
			x0 = a.div(a.mul(x0, priceScale), *Precision)
		}
		if a.err != nil {
			return zero, a.err
		}
		x1 := xp[i]
		xp[i] = x0
		D, err := newtonD(AGamma[0], AGamma[1], xp)
		if err != nil {
			return zero, err
		}
		t.D = D
		xp[i] = x1
		if now >= ti {
			t.FutureAGammaTime = 1
		}
	}

	temp, err := newtonY(AGamma[0], AGamma[1], xp, t.D, j)
	if err != nil {
		return zero, err
	}
	dy := a.sub(xp[j], temp)
	xp[j] = a.sub(xp[j], dy)
	dy = a.sub(dy, one)
	if j > 0 {
		dy = a.div(a.mul(dy, *Precision), priceScale)
	}
	dy = a.div(dy, t.Precisions[j])
	if a.err != nil {
		return zero, a.err
	}
	f, err := t.feeCalc(xp)
	if err != nil {
		return zero, err
	}
	dy = a.sub(dy, a.div(a.mul(f, dy), tenPow10))
	// assert dy >= min_dy, "Slippage"
	y = a.sub(y, dy)
	if a.err != nil {
		return zero, a.err
	}
	t.Info.Reserves[j] = y.ToBig()
	y = a.mul(y, t.Precisions[j])
	if j > 0 {
		y = a.div(a.mul(y, priceScale), *Precision)
	}
	xp[j] = y

	if dx.Gt(&tenPow5) && dy.Gt(&tenPow5) {
		_dx := a.mul(dx, t.Precisions[i])
		_dy := a.mul(dy, t.Precisions[j])
		// With two coins one of i, j is 0, so the "i != 0 and j != 0" branch of the N-coin code cannot happen.
		if i == 0 {
			p = a.div(a.mul(_dx, bone), _dy)
		} else {
			p = a.div(a.mul(_dy, bone), _dx)
			ix = i
		}
	}
	if a.err != nil {
		return zero, a.err
	}
	return dy, t.tweakPrice(AGamma, xp, ix, p, zero, now)
}

func (t *PoolSimulator) tweakPrice(AGamma [2]uint256.Int, _xp [nCoins]uint256.Int, i int, pI, newD uint256.Int,
	now int64) error {
	var a arith
	priceOracle := t.PriceOracle
	lastPrices := t.LastPrices

	if t.LastPricesTimestamp < now {
		var dt uint256.Int
		dt.SetUint64(uint64(now - t.LastPricesTimestamp))
		alpha, err := halfpow(a.div(a.mul(dt, bone), t.MaHalfTime), tenPow10)
		if a.err != nil {
			return a.err
		}
		if err != nil {
			return err
		}
		priceOracle = a.div(a.add(a.mul(lastPrices, a.sub(bone, alpha)), a.mul(priceOracle, alpha)), bone)
		if a.err != nil {
			return a.err
		}
		t.PriceOracle = priceOracle
		t.LastPricesTimestamp = now
	}

	DUnadjusted := newD
	if newD.IsZero() {
		var err error
		if DUnadjusted, err = newtonD(AGamma[0], AGamma[1], _xp); err != nil {
			return err
		}
	}
	priceScale := t.PriceScale

	if !pI.IsZero() {
		if i > 0 {
			lastPrices = pI
		} else {
			lastPrices = a.div(a.mul(lastPrices, bone), pI)
		}
	} else {
		xp := _xp
		dxPrice := a.div(xp[0], tenPow6)
		xp[0] = a.add(xp[0], dxPrice)
		temp, err := newtonY(AGamma[0], AGamma[1], xp, DUnadjusted, 1)
		if err != nil {
			return err
		}
		lastPrices = a.div(a.mul(priceScale, dxPrice), a.sub(_xp[1], temp))
	}
	if a.err != nil {
		return a.err
	}
	t.LastPrices = lastPrices

	totalSupply := t.LpSupply
	oldXcpProfit := t.XcpProfit
	oldVirtualPrice := t.VirtualPrice

	var xp [nCoins]uint256.Int
	xp[0] = a.div(DUnadjusted, nCoinsU)
	xp[1] = a.div(a.mul(DUnadjusted, bone), a.mul(nCoinsU, priceScale))
	xcpProfit := bone
	virtualPrice := bone
	if a.err != nil {
		return a.err
	}

	if !oldVirtualPrice.IsZero() {
		xcp, err := geometricMean(xp, true)
		if err != nil {
			return err
		}
		virtualPrice = a.div(a.mul(bone, xcp), totalSupply)
		xcpProfit = a.div(a.mul(oldXcpProfit, virtualPrice), oldVirtualPrice)
		if a.err != nil {
			return a.err
		}
		aGammaTime := t.FutureAGammaTime
		if virtualPrice.Lt(&oldVirtualPrice) && aGammaTime == 0 {
			return errLoss
		}
		if aGammaTime == 1 {
			t.FutureAGammaTime = 0
		}
	}
	t.XcpProfit = xcpProfit

	needsAdjustment := t.NotAdjusted
	// virtual_price * 2 - 10**18 > xcp_profit + 2 * allowed_extra_profit; the left side is negative (so the
	// condition is false) when virtual_price * 2 < 10**18.
	if vp2 := a.mul(virtualPrice, two); !vp2.Lt(&bone) {
		lhs := a.sub(vp2, bone)
		rhs := a.add(xcpProfit, a.mul(two, t.AllowedExtraProfit))
		if a.err != nil {
			return a.err
		}
		if lhs.Gt(&rhs) {
			needsAdjustment = true
			t.NotAdjusted = true
		}
	}

	if needsAdjustment {
		adjustmentStep := t.AdjustmentStep
		ratio := a.div(a.mul(priceOracle, bone), priceScale)
		if ratio.Gt(&bone) {
			ratio = a.sub(ratio, bone)
		} else {
			ratio = a.sub(bone, ratio)
		}
		norm := a.mul(ratio, ratio)
		stepSq := a.mul(adjustmentStep, adjustmentStep)
		if a.err != nil {
			return a.err
		}
		if norm.Gt(&stepSq) && !oldVirtualPrice.IsZero() {
			var err error
			if norm, err = sqrtInt(a.div(norm, bone)); err != nil {
				return err
			}
			pNew := a.div(a.add(a.mul(priceScale, a.sub(norm, adjustmentStep)), a.mul(adjustmentStep, priceOracle)), norm)
			xp = _xp
			xp[1] = a.div(a.mul(_xp[1], pNew), priceScale)
			if a.err != nil {
				return a.err
			}
			D, err := newtonD(AGamma[0], AGamma[1], xp)
			if err != nil {
				return err
			}
			xp[0] = a.div(D, nCoinsU)
			xp[1] = a.div(a.mul(D, bone), a.mul(nCoinsU, pNew))
			if a.err != nil {
				return a.err
			}
			gm, err := geometricMean(xp, true)
			if err != nil {
				return err
			}
			oldVirtualPrice = a.div(a.mul(bone, gm), totalSupply)
			if a.err != nil {
				return a.err
			}
			if oldVirtualPrice.Gt(&bone) {
				if lhs := a.sub(a.mul(two, oldVirtualPrice), bone); a.err == nil && lhs.Gt(&xcpProfit) {
					t.PriceScale = pNew
					t.D = D
					t.VirtualPrice = oldVirtualPrice
					return nil
				}
				if a.err != nil {
					return a.err
				}
			}
			t.NotAdjusted = false
		}
	}
	t.D = DUnadjusted
	t.VirtualPrice = virtualPrice
	return nil
}

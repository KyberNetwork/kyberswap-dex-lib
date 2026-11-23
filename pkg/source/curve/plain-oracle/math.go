package plainoracle

import (
	"time"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

/**
  Solidity code: https://optimistic.etherscan.io/address/0xb90b9b1f91a01ea22a182cd84c1e22222e39b415#code
*/

func _xpMem(
	balances []*uint256.Int,
	rates []*uint256.Int,
) ([]*uint256.Int, error) {
	var numTokens = len(balances)
	if numTokens != len(rates) {
		return nil, ErrBalancesMustMatchMultipliers
	}
	xp := make([]*uint256.Int, numTokens)
	for i := range numTokens {
		xp[i] = new(uint256.Int).Mul(rates[i], balances[i])
		xp[i].Div(xp[i], Precision)
	}
	return xp, nil
}

func (t *PoolSimulator) _xp() []*uint256.Int {
	var nTokens = len(t.Info.Tokens)
	result := make([]*uint256.Int, nTokens)
	for i := range nTokens {
		result[i] = new(uint256.Int).Mul(t.Rates[i], t.Info.Reserves[i])
		result[i].Div(result[i], Precision)
	}
	return result
}

func (t *PoolSimulator) get_D_mem(balances []*uint256.Int, amp *uint256.Int) (*uint256.Int, error) {
	var xp, err = _xpMem(balances, t.Rates)
	if err != nil {
		return nil, err
	}
	return t.getD(xp, amp)
}

func (t *PoolSimulator) _A() *uint256.Int {
	var t1 = t.FutureATime
	var a1 = t.FutureA
	var now = time.Now().Unix()
	if t1 > now {
		var t0 = t.InitialATime
		var a0 = t.InitialA
		var span = uint256.NewInt(uint64(t1 - t0))
		var elapsed = uint256.NewInt(uint64(now - t0))
		var res uint256.Int
		if a1.Cmp(a0) > 0 {
			var step uint256.Int
			step.Sub(a1, a0)
			step.Mul(&step, elapsed)
			step.Div(&step, span)
			return res.Add(a0, &step)
		}
		var step uint256.Int
		step.Sub(a0, a1)
		step.Mul(&step, elapsed)
		step.Div(&step, span)
		return res.Sub(a0, &step)
	}
	return a1
}

// getD returns the invariant D of xp. A zero sum returns zero, as the legacy simulator does.
func (t *PoolSimulator) getD(xp []*uint256.Int, a *uint256.Int) (*uint256.Int, error) {
	var numTokens = len(xp)
	var s uint256.Int
	for i := range numTokens {
		s.Add(&s, xp[i])
	}
	if s.IsZero() {
		return &s, nil
	}
	var numTokensU256 = uint256.NewInt(uint64(numTokens))
	var d, prevD, dP, nA, tmp, denom, num uint256.Int
	d.Set(&s)
	nA.Mul(a, numTokensU256)
	for range MaxLoopLimit {
		dP.Set(&d)
		for j := range numTokens {
			// +1 in the denominator prevents /0 (https://github.com/curvefi/curve-contract/blob/d4e8589/contracts/pools/aave/StableSwapAave.vy#L299)
			denom.Mul(xp[j], numTokensU256)
			denom.AddUint64(&denom, 1)
			dP.Mul(&dP, &d)
			dP.Div(&dP, &denom)
		}
		prevD.Set(&d)

		// d = (nA*s/A_PRECISION + dP*N) * d / ((nA - A_PRECISION)*d/A_PRECISION + dP*(N+1))
		tmp.Mul(&nA, &s)
		tmp.Div(&tmp, t.APrecision)
		num.Mul(&dP, numTokensU256)
		num.Add(&tmp, &num)
		num.Mul(&num, &d)

		var sub uint256.Int
		if nA.Lt(t.APrecision) {
			return nil, ErrDenominatorZero
		}
		sub.Sub(&nA, t.APrecision)
		tmp.Mul(&sub, &d)
		tmp.Div(&tmp, t.APrecision)
		denom.Mul(&dP, uint256.NewInt(uint64(numTokens+1)))
		denom.Add(&tmp, &denom)
		d.Div(&num, &denom)

		if withinDelta(&d, &prevD, 1) {
			return &d, nil
		}
	}
	return nil, ErrDDoesNotConverge
}

// withinDelta reports whether |x - y| <= delta.
func withinDelta(x, y *uint256.Int, delta uint64) bool {
	var diff uint256.Int
	if x.Cmp(y) > 0 {
		diff.Sub(x, y)
	} else {
		diff.Sub(y, x)
	}
	return diff.CmpUint64(delta) <= 0
}

func (t *PoolSimulator) getY(
	tokenIndexFrom int,
	tokenIndexTo int,
	x *uint256.Int,
	xp []*uint256.Int,
	dCached *uint256.Int,
) (*uint256.Int, error) {
	var numTokens = len(xp)
	if tokenIndexFrom == tokenIndexTo {
		return nil, ErrTokenFromEqualsTokenTo
	}
	if tokenIndexFrom >= numTokens && tokenIndexTo >= numTokens {
		return nil, ErrTokenIndexesOutOfRange
	}
	var numTokensU256 = uint256.NewInt(uint64(numTokens))
	var a = t._A()
	if a == nil {
		return nil, ErrInvalidAValue
	}

	var d uint256.Int
	if dCached != nil {
		d.Set(dCached)
	} else {
		dd, err := t.getD(xp, a)
		if err != nil {
			return nil, err
		}
		d.Set(dd)
	}
	var c, s, nA, _x uint256.Int
	c.Set(&d)
	nA.Mul(a, numTokensU256)
	for i := range numTokens {
		if i == tokenIndexFrom {
			_x.Set(x)
		} else if i != tokenIndexTo {
			_x.Set(xp[i])
		} else {
			continue
		}
		if _x.IsZero() {
			return nil, ErrZero
		}
		s.Add(&s, &_x)
		c.Mul(&c, &d)
		c.Div(&c, new(uint256.Int).Mul(&_x, numTokensU256))
	}
	if nA.IsZero() {
		return nil, ErrZero
	}
	c.Mul(&c, &d)
	c.Mul(&c, t.APrecision)
	c.Div(&c, new(uint256.Int).Mul(&nA, numTokensU256))
	var b uint256.Int
	b.Mul(&d, t.APrecision)
	b.Div(&b, &nA)
	b.Add(&s, &b)

	var yPrev, y, denom uint256.Int
	y.Set(&d)
	for range MaxLoopLimit {
		yPrev.Set(&y)
		// denom = 2*y + b - d, must stay positive
		denom.Mul(&y, uint256.NewInt(2))
		denom.Add(&denom, &b)
		if denom.Cmp(&d) <= 0 {
			return nil, ErrDenominatorZero
		}
		denom.Sub(&denom, &d)
		y.Mul(&y, &y)
		y.Add(&y, &c)
		y.Div(&y, &denom)
		if withinDelta(&y, &yPrev, 1) {
			return &y, nil
		}
	}
	return nil, ErrAmountOutNotConverge
}

func (t *PoolSimulator) GetDy(
	i int,
	j int,
	dx *uint256.Int,
	dCached *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	var xp = t._xp()
	// x: uint256 = xp[i] + (dx * rates[i] / PRECISION)
	var x uint256.Int
	x.Mul(dx, t.Rates[i])
	x.Div(&x, Precision)
	x.Add(&x, xp[i])

	// y: uint256 = self.get_y(i, j, x, xp)
	y, err := t.getY(i, j, &x, xp, dCached)
	if err != nil {
		return nil, nil, err
	}

	// dy: uint256 = xp[j] - y; legacy returns a negative value here, which uint256 cannot hold
	if xp[j].Cmp(y) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var dy uint256.Int
	dy.Sub(xp[j], y)

	// fee: uint256 = self.fee * dy / FEE_DENOMINATOR
	var fee uint256.Int
	fee.Mul(t.Info.SwapFee, &dy)
	fee.Div(&fee, FeeDenominator)

	// (dy - fee) * PRECISION / rates[j]
	if dy.Cmp(&fee) < 0 {
		return nil, nil, ErrNegativeResult
	}
	dy.Sub(&dy, &fee)
	dy.Mul(&dy, Precision)
	dy.Div(&dy, t.Rates[j])

	// fee * PRECISION / rates[j]
	fee.Mul(&fee, Precision)
	fee.Div(&fee, t.Rates[j])
	return &dy, &fee, nil
}

func (t *PoolSimulator) getYD(
	a *uint256.Int,
	tokenIndex int,
	xp []*uint256.Int,
	d *uint256.Int,
) (*uint256.Int, error) {
	var numTokens = len(xp)
	if tokenIndex >= numTokens {
		return nil, ErrTokenNotFound
	}
	var numTokensU256 = uint256.NewInt(uint64(numTokens))
	var c, s, nA uint256.Int
	c.Set(d)
	nA.Mul(a, numTokensU256)
	for i := range numTokens {
		if i != tokenIndex {
			s.Add(&s, xp[i])
			c.Mul(&c, d)
			c.Div(&c, new(uint256.Int).Mul(xp[i], numTokensU256))
		}
	}
	if nA.IsZero() {
		return nil, ErrZero
	}
	c.Mul(&c, d)
	c.Mul(&c, t.APrecision)
	c.Div(&c, new(uint256.Int).Mul(&nA, numTokensU256))
	var b uint256.Int
	b.Mul(d, t.APrecision)
	b.Div(&b, &nA)
	b.Add(&s, &b)

	var yPrev, y, denom uint256.Int
	y.Set(d)
	for range MaxLoopLimit {
		yPrev.Set(&y)
		denom.Mul(&y, uint256.NewInt(2))
		denom.Add(&denom, &b)
		if denom.Cmp(d) <= 0 {
			return nil, ErrDenominatorZero
		}
		denom.Sub(&denom, d)
		y.Mul(&y, &y)
		y.Add(&y, &c)
		y.Div(&y, &denom)
		if withinDelta(&y, &yPrev, 1) {
			return &y, nil
		}
	}
	return nil, ErrAmountOutNotConverge
}

func (t *PoolSimulator) CalculateWithdrawOneCoin(
	tokenAmount *uint256.Int,
	i int,
) (*uint256.Int, *uint256.Int, error) {
	var amp = t._A()
	var xp = t._xp()
	D0, err := t.getD(xp, amp)
	if err != nil {
		return nil, nil, err
	}
	// D1 = D0 - tokenAmount * D0 / totalSupply; legacy goes negative here (signed getYD), uint256 cannot hold it
	var sub uint256.Int
	sub.Mul(tokenAmount, D0)
	sub.Div(&sub, t.LpSupply)
	if D0.Cmp(&sub) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var D1 uint256.Int
	D1.Sub(D0, &sub)
	newY, err := t.getYD(amp, i, xp, &D1)
	if err != nil {
		return nil, nil, err
	}
	var nCoins = len(t.Info.Reserves)
	var xpReduced = make([]*uint256.Int, nCoins)
	var nCoinU256 = uint256.NewInt(uint64(nCoins))
	var fee uint256.Int
	fee.Mul(t.Info.SwapFee, nCoinU256)
	fee.Div(&fee, new(uint256.Int).Mul(big256.U4, new(uint256.Int).Sub(nCoinU256, big256.U1)))
	for j := range nCoins {
		var dxExpected, tmp uint256.Int
		tmp.Mul(xp[j], &D1)
		tmp.Div(&tmp, D0)
		if j == i {
			dxExpected.Sub(&tmp, newY)
		} else {
			dxExpected.Sub(xp[j], &tmp)
		}
		var reduce uint256.Int
		reduce.Mul(&fee, &dxExpected)
		reduce.Div(&reduce, FeeDenominator)
		xpReduced[j] = new(uint256.Int).Sub(xp[j], &reduce)
	}
	newYD, err := t.getYD(amp, i, xpReduced, &D1)
	if err != nil {
		return nil, nil, err
	}
	// legacy returns dy = (xpReduced[i] - newYD - 1) / multiplier and dyFee = dy0 - dy; either can be negative
	if xpReduced[i].Cmp(newYD) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var dy uint256.Int
	dy.Sub(xpReduced[i], newYD)
	if dy.IsZero() {
		return nil, nil, ErrNegativeResult
	}
	dy.Sub(&dy, big256.U1)
	dy.Div(&dy, t.Multipliers[i])
	if xp[i].Cmp(newY) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var dy0 uint256.Int
	dy0.Sub(xp[i], newY)
	dy0.Div(&dy0, t.Multipliers[i])
	if dy0.Cmp(&dy) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var dyFee uint256.Int
	dyFee.Sub(&dy0, &dy)
	return &dy, &dyFee, nil
}

func (t *PoolSimulator) CalculateTokenAmount(
	amounts []*uint256.Int,
	deposit bool,
) (*uint256.Int, error) {
	var numTokens = len(t.Info.Tokens)
	var a = t._A()
	d0, err := t.get_D_mem(t.Info.Reserves, a)
	if err != nil {
		return nil, err
	}
	var balances1 = make([]*uint256.Int, numTokens)
	for i := range numTokens {
		balances1[i] = new(uint256.Int)
		if deposit {
			balances1[i].Add(t.Info.Reserves[i], amounts[i])
		} else {
			if t.Info.Reserves[i].Cmp(amounts[i]) < 0 {
				return nil, ErrWithdrawMoreThanAvailable
			}
			balances1[i].Sub(t.Info.Reserves[i], amounts[i])
		}
	}
	d1, err := t.get_D_mem(balances1, a)
	if err != nil {
		return nil, err
	}
	// the legacy diff is d1 - d0 on deposit and d0 - d1 on withdraw; a wrong sign is an error, not a wrap
	var diff uint256.Int
	if deposit {
		if d1.Cmp(d0) < 0 {
			return nil, ErrD1LowerThanD0
		}
		diff.Sub(d1, d0)
	} else {
		if d0.Cmp(d1) < 0 {
			return nil, ErrD1LowerThanD0
		}
		diff.Sub(d0, d1)
	}
	diff.Mul(&diff, t.LpSupply)
	return diff.Div(&diff, d0), nil
}

func (t *PoolSimulator) AddLiquidity(amounts []*uint256.Int) (*uint256.Int, error) {
	var nCoins = len(amounts)
	var nCoinsU256 = uint256.NewInt(uint64(nCoins))
	var amp = t._A()
	// snapshot the live balances: AddLiquidity replaces t.Info.Reserves[i] below
	var oldBalances = make([]*uint256.Int, nCoins)
	for i := range nCoins {
		oldBalances[i] = new(uint256.Int).Set(t.Info.Reserves[i])
	}
	D0, err := t.get_D_mem(oldBalances, amp)
	if err != nil {
		return nil, err
	}
	var tokenSupply = t.LpSupply
	var newBalances = make([]*uint256.Int, nCoins)
	for i := range nCoins {
		newBalances[i] = new(uint256.Int).Add(oldBalances[i], amounts[i])
	}
	D1, err := t.get_D_mem(newBalances, amp)
	if err != nil {
		return nil, err
	}
	if D1.Cmp(D0) <= 0 {
		return nil, ErrD1LowerThanD0
	}
	var D2, mintAmount uint256.Int
	if tokenSupply.Sign() > 0 {
		var fee uint256.Int
		fee.Mul(t.Info.SwapFee, nCoinsU256)
		fee.Div(&fee, new(uint256.Int).Mul(big256.U4, uint256.NewInt(uint64(nCoins-1))))
		var adminFee = t.AdminFee
		for i := range nCoins {
			var ideal, difference, feeI, share uint256.Int
			ideal.Mul(D1, oldBalances[i])
			ideal.Div(&ideal, D0)
			if ideal.Cmp(newBalances[i]) > 0 {
				difference.Sub(&ideal, newBalances[i])
			} else {
				difference.Sub(newBalances[i], &ideal)
			}
			feeI.Mul(&fee, &difference)
			feeI.Div(&feeI, FeeDenominator)
			share.Mul(&feeI, adminFee)
			share.Div(&share, FeeDenominator)
			// legacy new_balances - share and new_balances - fee go negative when the subtrahend is larger
			if newBalances[i].Cmp(&share) < 0 {
				return nil, ErrNegativeResult
			}
			t.Info.Reserves[i] = new(uint256.Int).Sub(newBalances[i], &share)
			if newBalances[i].Cmp(&feeI) < 0 {
				return nil, ErrNegativeResult
			}
			newBalances[i] = new(uint256.Int).Sub(newBalances[i], &feeI)
		}
		D2Ptr, err := t.get_D_mem(newBalances, amp)
		if err != nil {
			return nil, err
		}
		D2.Set(D2Ptr)
		mintAmount.Sub(&D2, D0)
		mintAmount.Mul(&mintAmount, tokenSupply)
		mintAmount.Div(&mintAmount, D0)
	} else {
		for i := range nCoins {
			t.Info.Reserves[i] = new(uint256.Int).Set(newBalances[i])
		}
		mintAmount.Set(D1)
	}
	t.LpSupply = new(uint256.Int).Add(t.LpSupply, &mintAmount)
	return &mintAmount, nil
}

func (t *PoolSimulator) RemoveLiquidityOneCoin(tokenAmount *uint256.Int, i int) (*uint256.Int, error) {
	dy, dyFee, err := t.CalculateWithdrawOneCoin(tokenAmount, i)
	if err != nil {
		return nil, err
	}
	var admin uint256.Int
	admin.Mul(dyFee, t.AdminFee)
	admin.Div(&admin, FeeDenominator)
	var take uint256.Int
	take.Add(dy, &admin)
	// legacy lets the reserve and LP supply go negative; uint256 cannot hold that, so check before writing state
	if t.Info.Reserves[i].Cmp(&take) < 0 || t.LpSupply.Cmp(tokenAmount) < 0 {
		return nil, ErrNegativeResult
	}
	t.Info.Reserves[i] = new(uint256.Int).Sub(t.Info.Reserves[i], &take)
	t.LpSupply = new(uint256.Int).Sub(t.LpSupply, tokenAmount)
	return dy, nil
}

// GetVirtualPrice returns the virtual price and D.
func (t *PoolSimulator) GetVirtualPrice() (*uint256.Int, *uint256.Int, error) {
	var xp = t._xp()
	var A = t._A()
	D, err := t.getD(xp, A)
	if err != nil {
		return nil, nil, err
	}
	if t.LpSupply.IsZero() {
		return nil, nil, ErrDenominatorZero
	}
	var vPrice uint256.Int
	vPrice.Mul(D, Precision)
	vPrice.Div(&vPrice, t.LpSupply)
	return &vPrice, D, nil
}

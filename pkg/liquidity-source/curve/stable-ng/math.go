package stableng

import (
	"fmt"
	"math"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
)

func XpMem(
	rates []uint256.Int,
	balances []uint256.Int,
) []uint256.Int {
	// try to put `result` in caller's stack (this func will be inlined)
	var result [shared.MaxTokenCount]uint256.Int
	count := xpMem_inplace(rates, balances, result[:])
	return result[:count]
}

func xpMem_inplace(
	rates []uint256.Int,
	balances []uint256.Int,
	xp []uint256.Int,
) int {
	numTokens := len(rates)
	for i := 0; i < numTokens; i++ {
		xp[i].Div(number.SafeMul(&rates[i], &balances[i]), Precision)
	}
	return numTokens
}

func (t *PoolSimulator) _A() *uint256.Int {
	var t1 = t.Extra.FutureATime
	var a1 = t.Extra.FutureA
	var now = time.Now().Unix()
	if t1 > now {
		var t0 = t.Extra.InitialATime
		var a0 = t.Extra.InitialA
		if a1.Cmp(a0) > 0 {
			return number.Add(
				a0,
				number.Div(
					number.Mul(
						number.Sub(a1, a0),
						number.SetUint64(uint64(now-t0)),
					),
					number.SetUint64(uint64(t1-t0)),
				),
			)
		} else {
			return number.Sub(
				a0,
				number.Div(
					number.Mul(
						number.Sub(a0, a1),
						number.SetUint64(uint64(now-t0)),
					),
					number.SetUint64(uint64(t1-t0)),
				),
			)
		}
	}
	return a1
}

// D invariant calculation in non-overflowing integer operations iteratively
// - `D`: output
func (t *PoolSimulator) getD(xp []uint256.Int, a *uint256.Int, D *uint256.Int) error {
	var S uint256.Int
	S.Clear()
	for i := range xp {
		if xp[i].IsZero() {
			// this will cause div by zero down below
			return ErrZero
		}
		S.Add(&S, &xp[i])
	}
	if S.IsZero() {
		D.Clear()
		return nil
	}

	var D_P, Ann, Ann_mul_S_div_APrec, Ann_sub_APrec, Dprev uint256.Int

	// D: uint256 = S
	D.Set(&S)

	// Ann: uint256 = amp * N_COINS
	Ann.Mul(a, &t.NumTokensU256)

	// pre-calculate some values to be used in the loop
	// Ann * S / A_PRECISION
	Ann_mul_S_div_APrec.Div(number.Mul(&Ann, &S), t.StaticExtra.APrecision)
	// Ann - A_PRECISION
	Ann_sub_APrec.Sub(&Ann, t.StaticExtra.APrecision)

	numTokensPlus1 := uint256.NewInt(uint64(t.NumTokens + 1))
	numTokensPow := uint256.NewInt(uint64(math.Pow(float64(t.NumTokens), float64(t.NumTokens))))

	for i := 0; i < MaxLoopLimit; i++ {
		// D_P: uint256 = D
		D_P.Set(D)

		// for x in _xp: D_P = D_P * D / x
		for j := range xp {
			D_P.Div(
				number.SafeMul(&D_P, D),
				&xp[j],
			)
		}
		// D_P /= pow_mod256(N_COINS, N_COINS)
		D_P.Div(&D_P, numTokensPow)

		// Dprev = D
		Dprev.Set(D)

		// D = (Ann * S / A_PRECISION + D_P * N_COINS) * D / ((Ann - A_PRECISION) * D / A_PRECISION + (N_COINS + 1) * D_P)
		D.Div(
			number.SafeMul(
				number.SafeAdd(&Ann_mul_S_div_APrec, number.SafeMul(&D_P, &t.NumTokensU256)),
				D,
			),
			number.SafeAdd(
				number.Div(number.SafeMul(&Ann_sub_APrec, D), t.StaticExtra.APrecision),
				number.SafeMul(&D_P, numTokensPlus1),
			),
		)

		// calc abs(D - Dprev) and compare against 1
		if number.WithinDelta(D, &Dprev, 1) {
			return nil
		}
	}
	return ErrDDoesNotConverge
}

// Calculate x[j] if one makes x[i] = x
// - `dCached`: if `D` has been calculated before the reuse it here (use nil if not available)
// - `y`: output x[j]
func (t *PoolSimulator) GetY(
	tokenIndexFrom int,
	tokenIndexTo int,
	x *uint256.Int,
	xp []uint256.Int,
	dCached *uint256.Int,
	y *uint256.Int,
) error {
	if tokenIndexFrom == tokenIndexTo {
		return ErrTokenFromEqualsTokenTo
	}
	if tokenIndexFrom >= t.NumTokens && tokenIndexTo >= t.NumTokens {
		return ErrTokenIndexesOutOfRange
	}

	var a = t._A()
	if a == nil {
		return ErrInvalidAValue
	}

	var d uint256.Int
	if dCached != nil {
		d.Set(dCached)
	} else {
		err := t.getD(xp, a, &d)
		if err != nil {
			return err
		}
	}
	var c = number.Set(&d)
	var Ann = number.Mul(a, &t.NumTokensU256)
	var _x, s uint256.Int
	s.Clear()
	for i := 0; i < t.NumTokens; i++ {
		if i == tokenIndexFrom {
			_x.Set(x)
		} else if i != tokenIndexTo {
			_x.Set(&xp[i])
		} else {
			continue
		}
		if _x.IsZero() {
			return ErrZero
		}
		s.Add(&s, &_x)
		c.Div(
			number.SafeMul(c, &d),
			number.SafeMul(&_x, &t.NumTokensU256),
		)
	}
	if Ann.IsZero() {
		return ErrZero
	}
	c.Div(
		number.SafeMul(number.SafeMul(c, &d), t.StaticExtra.APrecision),
		number.SafeMul(Ann, &t.NumTokensU256),
	)
	var b = number.SafeAdd(
		&s,
		number.Div(number.SafeMul(&d, t.StaticExtra.APrecision), Ann),
	)

	var yPrev uint256.Int
	y.Set(&d)
	for i := 0; i < MaxLoopLimit; i++ {
		// y_prev = y
		yPrev.Set(y)

		// y = (y*y + c) / (2 * y + b - D)
		y.Div(
			number.SafeAdd(number.SafeMul(y, y), c),
			number.SafeSub(
				number.SafeAdd(
					number.SafeAdd(y, y), // 2 * y
					b),
				&d),
		)

		// calc abs(y - y_prev) and compare against 1
		if number.WithinDelta(y, &yPrev, 1) {
			return nil
		}
	}
	return ErrAmountOutNotConverge
}

// Calculate the current output dy given input dx
func (t *PoolSimulator) GetDy(
	i int,
	j int,
	dx *uint256.Int,
	dCached *uint256.Int,
	dy *uint256.Int,
	adminFee *uint256.Int,
) error {
	var xp = XpMem(t.Extra.RateMultipliers, t.Reserves)
	// x: uint256 = xp[i] + (dx * rates[i] / PRECISION)
	var x = number.SafeAdd(&xp[i], number.Div(number.SafeMul(dx, &t.Extra.RateMultipliers[i]), Precision))

	return t.GetDyByX(i, j, x, xp, dCached, dy, adminFee)
}

// Calculate the current output dy if already have `x` input
func (t *PoolSimulator) GetDyByX(
	i int,
	j int,
	x *uint256.Int,
	xp []uint256.Int,
	dCached *uint256.Int,
	dy *uint256.Int,
	adminFee *uint256.Int,
) error {
	// y: uint256 = self.get_y(i, j, x, xp)
	var y uint256.Int
	var err = t.GetY(i, j, x, xp, dCached, &y)
	if err != nil {
		return err
	}

	// dy: uint256 = _xp[j] - y - 1  # -1 just in case there were some rounding errors
	number.SafeSubZ(&xp[j], &y, dy)
	if dy.Sign() <= 0 {
		return ErrZero
	}
	dy.SubUint64(dy, 1)

	// dy_fee: uint256 = unsafe_div(
	//   dy * self._dynamic_fee(
	//      unsafe_div(_xp[i] + x, 2), unsafe_div(_xp[j] + y, 2), self.fee
	//   ),
	//   FEE_DENOMINATOR
	// )
	var dynamicFee, dyFee uint256.Int
	t.DynamicFee(
		number.Div(number.SafeAdd(&xp[i], x), number.Number_2),
		number.Div(number.SafeAdd(&xp[j], &y), number.Number_2),
		t.Extra.SwapFee,
		&dynamicFee,
	)
	dyFee.Div(
		number.SafeMul(dy, &dynamicFee),
		FeeDenominator,
	)

	// # Convert all to real units
	// dy = (dy - dy_fee) * PRECISION / rates[j]
	dy.Div(number.SafeMul(number.SafeSub(dy, &dyFee), Precision), &t.Extra.RateMultipliers[j])

	adminFee.Div(
		number.SafeMul(
			number.Div(
				number.SafeMul(&dyFee, t.Extra.AdminFee),
				FeeDenominator,
			),
			Precision,
		),
		&t.Extra.RateMultipliers[j],
	)

	return nil
}

// GetDx calculates the required input dx given output dy
// https://github.com/curvefi/stableswap-ng/blob/12a0c7df1fc490ff8e5a977a0cbadf86f1351c8f/contracts/main/CurveStableSwapNGViews.vy#L44
func (t *PoolSimulator) GetDx(
	i int,
	j int,
	dy *uint256.Int,
	dCached *uint256.Int,
	dx *uint256.Int,
	adminFee *uint256.Int,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if recoveredError, ok := r.(error); ok {
				err = errors.Wrapf(ErrExecutionReverted, "recovered error: %v", recoveredError)
			} else {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
		}
	}()

	var xp = XpMem(t.Extra.RateMultipliers, t.Reserves)

	// dy_with_fee: uint256 = dy * rates[j] / PRECISION + 1
	dyWithFee := number.Add(number.Div(number.SafeMul(dy, &t.Extra.RateMultipliers[j]), Precision), number.Number_1)

	var dynamicFee uint256.Int
	t.DynamicFee(
		&xp[i],
		&xp[j],
		t.Extra.SwapFee,
		&dynamicFee,
	)

	adminFee.Set(&dynamicFee)

	// y: uint256 = xp[j] - dy_with_fee * FEE_DENOMINATOR / (FEE_DENOMINATOR - fee)
	var y uint256.Int
	y.Sub(
		&xp[j],
		number.Div(
			number.SafeMul(dyWithFee, FeeDenominator),
			number.SafeSub(FeeDenominator, &dynamicFee),
		),
	)

	// x: uint256 = self.get_y(j, i, y, xp, amp, D, N_COINS)
	var x uint256.Int
	err = t.GetY(j, i, &y, xp, dCached, &x)
	if err != nil {
		return err
	}

	// return (x - xp[i]) * PRECISION / rates[i]
	dx.Div(number.SafeMul(number.SafeSub(&x, &xp[i]), Precision), &t.Extra.RateMultipliers[i])

	return nil
}

func (t *PoolSimulator) DynamicFee(xpi *uint256.Int, xpj *uint256.Int, swapFee *uint256.Int, feeOutput *uint256.Int) {
	_offpeg_fee_multiplier := t.StaticExtra.OffpegFeeMultiplier
	if _offpeg_fee_multiplier.Cmp(FeeDenominator) <= 0 {
		feeOutput.Set(swapFee)
		return
	}

	// xps2: uint256 = (xpi + xpj) ** 2
	sum := number.SafeAdd(xpi, xpj)
	prod := number.SafeMul(xpi, xpj)
	xps2 := number.SafeMul(sum, sum)
	feeOutput.Div(
		number.Mul(_offpeg_fee_multiplier, swapFee),
		number.Add(
			number.Div(
				number.SafeMul(
					number.SafeMul(
						number.Sub(_offpeg_fee_multiplier, FeeDenominator),
						number.Number_4,
					),
					prod,
				), xps2,
			),
			FeeDenominator,
		),
	)
}

// Calculate addition or reduction in token supply from a deposit or withdrawal
func (t *PoolSimulator) CalculateTokenAmountU256(
	amounts []uint256.Int,
	deposit bool,

	// output
	mintAmount *uint256.Int,
	feeAmounts []uint256.Int,
) error {
	var a = t._A()
	var d0, d1, d2 uint256.Int
	var xp = XpMem(t.Extra.RateMultipliers, t.Reserves)

	// Initial invariant
	err := t.getD(xp, a, &d0)
	if err != nil {
		return err
	}

	var newBalances [shared.MaxTokenCount]uint256.Int
	for i := 0; i < t.NumTokens; i++ {
		if deposit {
			number.SafeAddZ(&t.Reserves[i], &amounts[i], &newBalances[i])
		} else {
			number.SafeSubZ(&t.Reserves[i], &amounts[i], &newBalances[i])
		}
	}

	// Invariant after change
	xp = XpMem(t.Extra.RateMultipliers, newBalances[:t.NumTokens])
	err = t.getD(xp, a, &d1)
	if err != nil {
		return err
	}

	// We need to recalculate the invariant accounting for fees
	// to calculate fair user's share
	var totalSupply = &t.LpSupply
	if !totalSupply.IsZero() {
		// Only account for fees if we are not the first to deposit
		var baseFee = number.Div(
			number.Mul(t.Extra.SwapFee, &t.NumTokensU256),
			uint256.NewInt(4*uint64(t.NumTokens-1)),
		)
		var _dynamic_fee_i, difference, xs, ys uint256.Int
		// ys: uint256 = (D0 + D1) / N_COINS
		ys.Div(number.SafeAdd(&d0, &d1), &t.NumTokensU256)
		for i := 0; i < t.NumTokens; i++ {
			// ideal_balance: uint256 = D1 * old_balances[i] / D0
			ideal_balance := number.Div(number.SafeMul(&d1, &t.Reserves[i]), &d0)
			if ideal_balance.Cmp(&newBalances[i]) > 0 {
				difference.Sub(ideal_balance, &newBalances[i])
			} else {
				difference.Sub(&newBalances[i], ideal_balance)
			}

			// xs = old_balances[i] + new_balance
			number.SafeAddZ(&t.Reserves[i], &newBalances[i], &xs)

			// this line is from `add_liquidity` method, the `calc_token_amount` method doesn't have it (might be a bug)
			// xs = unsafe_div(rates[i] * (old_balances[i] + new_balance), PRECISION)
			xs.Div(number.SafeMul(&t.Extra.RateMultipliers[i], &xs), Precision)

			// _dynamic_fee_i = self._dynamic_fee(xs, ys, base_fee, fee_multiplier)
			t.DynamicFee(&xs, &ys, baseFee, &_dynamic_fee_i)

			// new_balances[i] -= _dynamic_fee_i * difference / FEE_DENOMINATOR
			fee := number.Div(number.SafeMul(&_dynamic_fee_i, &difference), FeeDenominator)
			number.SafeSubZ(&newBalances[i], fee, &newBalances[i])

			// record fee so we can update balance later
			// self.admin_balances[i] += unsafe_div(fees[i] * admin_fee, FEE_DENOMINATOR)
			feeAmounts[i].Div(number.SafeMul(fee, t.Extra.AdminFee), FeeDenominator)
		}

		for i := 0; i < t.NumTokens; i++ {
			// xp[idx] = rates[idx] * new_balances[idx] / PRECISION
			xp[i].Div(number.SafeMul(&t.Extra.RateMultipliers[i], &newBalances[i]), Precision)
		}
		// D2 = self.get_D(xp, amp, N_COINS)
		err = t.getD(xp, a, &d2)
		if err != nil {
			return err
		}
	} else {
		// Take the dust if there was any
		mintAmount.Set(&d1)
		return nil
	}

	var diff uint256.Int
	if deposit {
		number.SafeSubZ(&d2, &d0, &diff)
	} else {
		number.SafeSubZ(&d0, &d2, &diff)
	}
	// return diff * total_supply / D0
	mintAmount.Div(number.Mul(&diff, totalSupply), &d0)
	return nil
}

func (t *PoolSimulator) CalculateWithdrawOneCoinU256(tokenAmount *uint256.Int, i int, dy *uint256.Int, dyFee *uint256.Int) error {
	var amp = t._A()
	var xp = XpMem(t.Extra.RateMultipliers, t.Reserves)

	// First, need to calculate
	// * Get current D
	// * Solve Eqn against y_i for D - _token_amount
	var D0, D1, newY, newYD uint256.Int
	err := t.getD(xp, amp, &D0)
	if err != nil {
		return err
	}
	var totalSupply = &t.LpSupply
	// D1: uint256 = D0 - _burn_amount * D0 / total_supply
	number.SafeSubZ(&D0, number.Div(number.SafeMul(tokenAmount, &D0), totalSupply), &D1)
	err = t.getYD(amp, i, xp, &D1, &newY)
	if err != nil {
		return err
	}

	var baseFee = number.Div(
		number.Mul(t.Extra.SwapFee, &t.NumTokensU256),
		number.Mul(number.Number_4, uint256.NewInt(uint64(t.NumTokens-1))),
	)
	var xpReduced [shared.MaxTokenCount]uint256.Int
	// ys: uint256 = unsafe_div((D0 + D1), unsafe_mul(2, N_COINS))
	var ys = number.Div(number.SafeAdd(&D0, &D1), uint256.NewInt(uint64(t.NumTokens*2)))

	var dxExpected, xavg, dynamicFee uint256.Int
	for j := 0; j < t.NumTokens; j += 1 {
		if j == i {
			// dx_expected = xp_j * D1 / D0 - new_y
			number.SafeSubZ(number.Div(number.SafeMul(&xp[j], &D1), &D0), &newY, &dxExpected)
			// xavg = unsafe_div((xp_j + new_y), 2)
			xavg.Div(number.SafeAdd(&xp[j], &newY), number.Number_2)
		} else {
			// dx_expected = xp_j - xp_j * D1 / D0
			number.SafeSubZ(&xp[j], number.Div(number.SafeMul(&xp[j], &D1), &D0), &dxExpected)
			// xavg = xp_j
			xavg.Set(&xp[j])
		}

		// dynamic_fee = self._dynamic_fee(xavg, ys, base_fee)
		t.DynamicFee(&xavg, ys, baseFee, &dynamicFee)

		// xp_reduced[j] = xp_j - unsafe_div(dynamic_fee * dx_expected, FEE_DENOMINATOR)
		number.SafeSubZ(&xp[j], number.Div(number.SafeMul(&dynamicFee, &dxExpected), FeeDenominator), &xpReduced[j])
	}

	// dy: uint256 = xp_reduced[i] - self.get_y_D(amp, i, xp_reduced, D1)
	err = t.getYD(amp, i, xpReduced[:t.NumTokens], &D1, &newYD)
	if err != nil {
		return err
	}
	number.SafeSubZ(&xpReduced[i], &newYD, dy)

	// dy_0: uint256 = (xp[i] - new_y) * PRECISION / rates[i]  # w/o fees
	var dy0 = number.Div(number.SafeMul(number.SafeSub(&xp[i], &newY), Precision), &t.Extra.RateMultipliers[i])
	if dy.Sign() <= 0 {
		return ErrZero
	}
	// dy = unsafe_div((dy - 1) * PRECISION, rates[i])  # Withdraw less to account for rounding errors
	dy.Div(number.SafeMul(number.SafeSub(dy, number.Number_1), Precision), &t.Extra.RateMultipliers[i])

	number.SafeSubZ(dy0, dy, dyFee)

	return nil
}

// Calculate x[i] if one reduces D from being calculated for xp to D
func (t *PoolSimulator) getYD(
	a *uint256.Int,
	tokenIndex int,
	xp []uint256.Int,
	d *uint256.Int,
	// output
	y *uint256.Int,
) error {
	if tokenIndex >= t.NumTokens {
		return ErrTokenIndexesOutOfRange
	}
	var c, s uint256.Int
	c.Set(d)
	s.Clear()
	var nA = number.Mul(a, &t.NumTokensU256)
	for i := 0; i < t.NumTokens; i++ {
		if i != tokenIndex {
			number.SafeAddZ(&s, &xp[i], &s)
			c.Div(
				number.SafeMul(&c, d),
				number.SafeMul(&xp[i], &t.NumTokensU256),
			)
		}
	}
	if nA.IsZero() {
		return ErrZero
	}
	c.Div(
		number.SafeMul(number.SafeMul(&c, d), t.StaticExtra.APrecision),
		number.SafeMul(nA, &t.NumTokensU256),
	)
	var b = number.SafeAdd(
		&s,
		number.Div(number.SafeMul(d, t.StaticExtra.APrecision), nA),
	)
	var yPrev uint256.Int
	y.Set(d)
	for i := 0; i < MaxLoopLimit; i++ {
		yPrev.Set(y)
		// y = (y*y + c) / (2 * y + b - D)
		y.Div(
			number.SafeAdd(
				number.SafeMul(y, y),
				&c,
			),
			number.SafeSub(
				number.SafeAdd(
					number.SafeAdd(y, y),
					b,
				),
				d,
			),
		)
		if number.WithinDelta(y, &yPrev, 1) {
			return nil
		}
	}
	return ErrAmountOutNotConverge
}

func (t *PoolSimulator) ApplyRemoveLiquidityOneCoinU256(i int, tokenAmount, dy, dyFee *uint256.Int) error {
	number.SafeSubZ(
		&t.Reserves[i],
		number.SafeAdd(dy, number.Div(number.SafeMul(dyFee, t.Extra.AdminFee), FeeDenominator)),
		&t.Reserves[i])
	number.FillBig(&t.Reserves[i], t.Info.Reserves[i]) // always sync back update to t.Info, will be removed later
	t.LpSupply.Sub(&t.LpSupply, tokenAmount)
	return nil
}

func (t *PoolSimulator) ApplyAddLiquidity(amounts, feeAmounts []uint256.Int, mintAmount *uint256.Int) error {
	for i := 0; i < t.NumTokens; i++ {
		number.SafeAddZ(&t.Reserves[i], &amounts[i], &t.Reserves[i])
		number.SafeSubZ(&t.Reserves[i], &feeAmounts[i], &t.Reserves[i])
		number.FillBig(&t.Reserves[i], t.Info.Reserves[i]) // always sync back update to t.Info, will be removed later
	}

	t.LpSupply.Add(&t.LpSupply, mintAmount)

	return nil
}

func (t *PoolSimulator) GetVirtualPriceU256(vPrice *uint256.Int, D *uint256.Int) error {
	var xp = XpMem(t.Extra.RateMultipliers, t.Reserves)
	var A = t._A()
	var err = t.getD(xp, A, D)
	if err != nil {
		return err
	}
	if t.LpSupply.IsZero() {
		return ErrZero
	}
	vPrice.Div(number.Mul(D, Precision), &t.LpSupply)
	return nil
}

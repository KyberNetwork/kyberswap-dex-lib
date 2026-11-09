package twocryptong

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var (
	// v3MinFee and v3MaxFee bound v3.0.0's _fee: 0.1 bps and 100%, in FEE_PRECISION (1e10).
	v3MinFee = uint256.NewInt(1e5)
	v3MaxFee = U_1e10
	// v3FeePrecision2 is FEE_PRECISION ** 2, the scale of reserved_profit_fraction * admin_fee.
	v3FeePrecision2 = new(uint256.Int).Mul(U_1e10, U_1e10)
	// v3OracleBandMargin is the share of price_oracle by which a policy target must clear the oracle band's edges
	// when the oracle is carried from a view read rather than from the pool's own cache (see tweakPriceV3).
	v3OracleBandMargin = uint256.NewInt(1e6)
)

// v3PostTradeTracked reports whether the tracker stored all v3.0.0 state tweak_price reads, so the state after a
// trade can be simulated (else it is refused, ErrStateAfterTrade).
func (t *PoolSimulator) v3PostTradeTracked() bool {
	e := &t.Extra
	return e.Version == VersionV3 && e.LpXcpProfit != nil && e.ReservedProfitFraction != nil && e.AdminFee != nil &&
		e.AdjustmentStepMin != nil && e.AdjustmentStepMax != nil && e.MaTime != nil && !e.MaTime.IsZero() &&
		e.TwocryptoPolicy != nil && e.DonationShares != nil && e.LpSupply != nil && !e.LpSupply.IsZero() &&
		e.VirtualPrice != nil && e.XcpProfit != nil && e.D != nil &&
		len(e.PriceScale) == NumTokens-1 && len(e.PriceOracle) == NumTokens-1 && len(e.LastPrices) == NumTokens-1
}

// clampV3Fee is the [MIN_FEE, MAX_FEE] bound of v3.0.0's _fee.
func clampV3Fee(fee *uint256.Int) {
	if fee.Lt(v3MinFee) {
		fee.Set(v3MinFee)
	} else if fee.Gt(v3MaxFee) {
		fee.Set(v3MaxFee)
	}
}

// exchangeV3 is the part of v3.0.0's _exchange after the amount out: the admin share of the fee leaves
// balances[j], D is solved again on the remaining balances, and tweak_price runs. xp holds the scaled balances
// after the trade without the admin share (GetDy's). It fills s, including AdminFee.
func (t *PoolSimulator) exchangeV3(A, gamma *uint256.Int, j int, dy, fee *uint256.Int, s *SwapInfo) error {
	e := &t.Extra
	// fee * reserved_profit_fraction * admin_fee: fee < 2**128 and both factors <= 1e10, far below 2**256.
	s.AdminFee.Mul(fee, e.ReservedProfitFraction)
	s.AdminFee.Mul(&s.AdminFee, e.AdminFee)
	s.AdminFee.Div(&s.AdminFee, v3FeePrecision2)

	y := number.SafeSub(number.SafeSub(&t.Reserves[j], dy), &s.AdminFee)
	y = number.SafeMul(y, &t.precisionMultipliers[j])
	if j > 0 {
		y.Div(number.SafeMul(y, &e.PriceScale[j-1]), Precision)
	}
	s.Xp[j].Set(y)

	D, err := newton_D(A, gamma, &s.K0, s.Xp[:], e.UseCustomMath)
	if err != nil {
		return err
	}
	return t.tweakPriceV3(A, gamma, s.Xp, D, s)
}

// tweakPriceV3 is v3.0.0's tweak_price (verified source of 0x4f52c3a8) with the pool not ramping (the only
// v3.0.0 state quoted). Against v2.1.0d:
//   - price_oracle caps last_prices both ways, to [price_scale / 2, 2 * price_scale];
//   - the virtual price may not drop below the one before the trade at the stored D (vp_preop) nor the stored one;
//   - lp_xcp_profit, the LP share of the profit, gates the rebalance and must hold after it;
//   - the target is the policy's get_price_scale, clamped to price_oracle +- 20%, and the step is
//     min(norm / 5, adjustment_step_max), taken only if above adjustment_step_min;
//   - the balances must stay under 1000:1, and the policy stores the result (update_pool_state).
//
// The pool updates price_oracle from its cache; the tracker reads the price_oracle() view instead, which this
// carries forward exactly only when it was read at last_timestamp. Otherwise a target near the oracle band's
// edges leaves the state after the trade unknown (V3Exact false), as does one equal to the oracle.
func (t *PoolSimulator) tweakPriceV3(A, gamma *uint256.Int, _xp [NumTokens]uint256.Int, D *uint256.Int,
	s *SwapInfo) error {
	e := &t.Extra
	now := nowUnix()
	priceScale := &e.PriceScale[0]

	var vpPreop uint256.Int
	vpPreop.Div(number.SafeMul(U_1e18, xcpV21d(e.D, priceScale)), e.LpSupply)

	oracleExact := true
	if e.LastTimestamp < now {
		if _, err := t.priceOracleAt(now, s.PriceOracle[:]); err != nil {
			return err
		}
		oracleExact = e.PriceOracleTimestamp == e.LastTimestamp
	} else {
		copy(s.PriceOracle[:], e.PriceOracle)
	}
	s.PriceOracleTimestamp = now

	if err := get_p(e.UseCustomMath, _xp, D, A, gamma, s.LastPrices[:]); err != nil {
		return err
	}
	s.LastPrices[0].Div(number.SafeMul(&s.LastPrices[0], priceScale), U_1e18)

	totalSupply := e.LpSupply
	donationShares := t.donationSharesAt(now, true)
	if !donationShares.Lt(totalSupply) {
		return ErrNegativeDonation
	}
	var lockedSupply uint256.Int
	lockedSupply.Sub(totalSupply, donationShares)

	xcp := xcpV21d(D, priceScale)
	scaledXcp := number.SafeMul(U_1e18, xcp)
	var virtualPrice uint256.Int
	virtualPrice.Div(scaledXcp, totalSupply)
	if virtualPrice.Lt(&vpPreop) || virtualPrice.Lt(e.VirtualPrice) {
		return ErrLoss // "virtual price decreased"
	}
	t.profitV3(&virtualPrice, s)

	var vpBoosted uint256.Int
	vpBoosted.Div(scaledXcp, &lockedSupply)
	if vpBoosted.Lt(&virtualPrice) {
		return ErrNegativeDonation
	}

	// Without a rebalance: price_scale kept, D and virtual_price from the trade.
	s.PriceScale[0].Set(priceScale)
	s.D.Set(D)
	s.VirtualPrice.Set(&virtualPrice)
	s.LpSupply.Set(totalSupply)
	s.DonationShares.Set(e.DonationShares)
	s.LastDonationReleaseTs = e.LastDonationReleaseTs
	s.LastTimestamp = now
	s.V3Exact = true

	xp := _xp
	if vpBoosted.Gt(&s.LpXcpProfit) && now > e.LastTimestamp {
		// xp becomes the rebalanced balances when the pool keeps the new price_scale: the ones it checks.
		if _, err := t.rebalanceV3(A, gamma, &xp, &virtualPrice, donationShares, oracleExact, now, s); err != nil {
			return err
		}
	}
	if err := assertBalanceXp(&xp); err != nil {
		return err
	}
	policy, err := e.TwocryptoPolicy.afterUpdate(now, &s.LastPrices[0], &s.PriceScale[0], &s.PriceOracle[0])
	if err != nil {
		return err
	}
	s.TwocryptoPolicy = policy
	return nil
}

// profitV3 moves xcp_profit by the virtual price's change and lp_xcp_profit by the LP part of a gain (the legacy
// gross reserve over the profit net of the admin share), into s.
func (t *PoolSimulator) profitV3(virtualPrice *uint256.Int, s *SwapInfo) {
	e := &t.Extra
	oldVirtualPrice := e.VirtualPrice
	s.XcpProfit.Set(e.XcpProfit)
	s.LpXcpProfit.Set(e.LpXcpProfit)
	var delta uint256.Int
	if virtualPrice.Gt(oldVirtualPrice) {
		s.XcpProfit.Add(&s.XcpProfit, delta.Sub(virtualPrice, oldVirtualPrice))
		if !s.XcpProfit.Gt(Precision) {
			return
		}
		base := e.XcpProfit
		if base.Lt(Precision) {
			base = Precision
		}
		var profit, num, den uint256.Int
		profit.Sub(&s.XcpProfit, base)
		// d_profit < 2**128 and the fractions <= 1e10 each.
		num.Mul(&profit, e.ReservedProfitFraction)
		num.Mul(&num, den.Sub(U_1e10, e.AdminFee))
		den.Sub(v3FeePrecision2, den.Mul(e.ReservedProfitFraction, e.AdminFee))
		s.LpXcpProfit.Add(&s.LpXcpProfit, num.Div(&num, &den))
		return
	}
	// Equal only: the trade may not lower the virtual price outside a ramp.
	delta.Sub(oldVirtualPrice, virtualPrice)
	s.XcpProfit.Sub(&s.XcpProfit, &delta)
	var headroom uint256.Int
	if s.LpXcpProfit.Gt(Precision) && !delta.Gt(headroom.Sub(&s.LpXcpProfit, Precision)) {
		s.LpXcpProfit.Sub(&s.LpXcpProfit, &delta)
	} else {
		s.LpXcpProfit.Set(Precision)
	}
}

// rebalanceV3 is the price_scale adjustment in v3.0.0's tweak_price. It writes s and the rebalanced xp only when
// the pool keeps the new price_scale, and reports whether it does.
func (t *PoolSimulator) rebalanceV3(A, gamma *uint256.Int, xp *[NumTokens]uint256.Int,
	virtualPrice, donationShares *uint256.Int, oracleExact bool, now int64, s *SwapInfo) (bool, error) {
	e := &t.Extra
	priceScale, priceOracle := &e.PriceScale[0], &s.PriceOracle[0]
	pPolicy, err := e.TwocryptoPolicy.PriceScaleAt(now)
	if err != nil {
		return false, err
	}

	target := priceOracle
	if pPolicy.IsZero() {
		s.V3Exact = s.V3Exact && oracleExact
	} else {
		var bound, lo, hi, margin uint256.Int
		bound.Div(priceOracle, number.Number_5)
		lo.Sub(priceOracle, &bound)
		hi.Add(priceOracle, &bound)
		target = pPolicy
		if target.Lt(&lo) {
			target = &lo
		} else if target.Gt(&hi) {
			target = &hi
		}
		if !oracleExact {
			margin.Div(priceOracle, v3OracleBandMargin)
			var loIn, hiIn uint256.Int
			loIn.Add(&lo, &margin)
			hiIn.Sub(&hi, &margin)
			if !pPolicy.Gt(&loIn) || !pPolicy.Lt(&hiIn) {
				s.V3Exact = false
			}
		}
	}

	var norm uint256.Int
	norm.Div(number.SafeMul(target, Precision), priceScale)
	if norm.Gt(U_1e18) {
		norm.Sub(&norm, U_1e18)
	} else {
		norm.Sub(U_1e18, &norm)
	}
	step := new(uint256.Int).Div(&norm, number.Number_5)
	if step.Gt(e.AdjustmentStepMax) {
		step.Set(e.AdjustmentStepMax)
	}
	if !step.Gt(e.AdjustmentStepMin) {
		return false, nil
	}
	var pNew, tmp uint256.Int
	pNew.Mul(priceScale, tmp.Sub(&norm, step))
	pNew.Add(&pNew, tmp.Mul(step, target))
	pNew.Div(&pNew, &norm)
	if pNew.Eq(priceScale) {
		return false, nil
	}

	newXp := [NumTokens]uint256.Int{xp[0]}
	newXp[1].Div(number.SafeMul(&xp[1], &pNew), priceScale)
	newD, err := newton_D(A, gamma, new(uint256.Int), newXp[:], e.UseCustomMath)
	if err != nil {
		return false, err
	}
	totalSupply := e.LpSupply
	scaledXcp := number.SafeMul(U_1e18, xcpV21d(newD, &pNew))
	var newVirtualPrice, burn uint256.Int
	newVirtualPrice.Div(scaledXcp, totalSupply)

	goal := &s.LpXcpProfit
	if virtualPrice.Gt(goal) {
		goal = virtualPrice
	}
	if newVirtualPrice.Lt(goal) {
		tweakedSupply := new(uint256.Int).Div(scaledXcp, goal)
		if !tweakedSupply.Lt(totalSupply) {
			return false, ErrTweakedSupply
		}
		burn.Sub(totalSupply, tweakedSupply)
		if donationShares.Lt(&burn) {
			burn.Set(donationShares)
		}
		newVirtualPrice.Div(scaledXcp, tmp.Sub(totalSupply, &burn))
	}
	if !newVirtualPrice.Gt(U_1e18) || newVirtualPrice.Lt(&s.LpXcpProfit) {
		return false, nil
	}

	s.PriceScale[0].Set(&pNew)
	s.D.Set(newD)
	s.VirtualPrice.Set(&newVirtualPrice)
	*xp = newXp
	if !burn.IsZero() {
		t.burnDonationShares(&burn, donationShares, now, s)
	}
	return true, nil
}

// burnDonationShares applies a rebalance's burn of donation shares to s: the supply drops, and the release
// timestamp moves so that the shares available after protection drop by the burn.
func (t *PoolSimulator) burnDonationShares(burn, donationShares *uint256.Int, now int64, s *SwapInfo) {
	e := &t.Extra
	unlocked := t.donationSharesAt(now, false)
	var unlockedNew, newTotal, newElapsed, tmp uint256.Int
	unlockedNew.Sub(unlocked, tmp.Div(tmp.Mul(burn, unlocked), donationShares))
	newTotal.Sub(e.DonationShares, burn)
	if !newTotal.IsZero() && !unlockedNew.IsZero() {
		newElapsed.Div(tmp.Mul(&unlockedNew, e.DonationDuration), &newTotal)
	}
	s.DonationShares.Set(&newTotal)
	s.LpSupply.Sub(e.LpSupply, burn)
	s.LastDonationReleaseTs = now - int64(newElapsed.Uint64())
}

// assertBalanceXp is v3.0.0's _assert_balance: both scaled balances positive and less than 1000:1.
func assertBalanceXp(xp *[NumTokens]uint256.Int) error {
	hi, lo := &xp[0], &xp[1]
	if hi.Lt(lo) {
		hi, lo = lo, hi
	}
	if lo.IsZero() || !new(uint256.Int).Div(hi, lo).Lt(uint256.NewInt(1000)) {
		return ErrUnbalanced
	}
	return nil
}

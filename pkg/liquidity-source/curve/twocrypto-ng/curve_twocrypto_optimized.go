package twocryptong

import (
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

// nowUnix stands for block.timestamp; tests replace it.
var nowUnix = func() int64 { return time.Now().Unix() }

// from contracts/main/CurveTwocryptoOptimized.vy
// (there is no CurveTwocryptoOptimizedWETH.vy)

func (t *PoolSimulator) FeeCalc(xp []uint256.Int, fee *uint256.Int) error {
	return t.feeCalc(xp, fee, true)
}

// feeCalc: with discount=false a policy pool pays its fallback fee, the most a swap pays.
func (t *PoolSimulator) feeCalc(xp []uint256.Int, fee *uint256.Int, discount bool) error {
	if p := t.Extra.Policy; p != nil {
		if !discount {
			fee.Set(p.FallbackFee)
		} else if err := p.fee(xp, t.Extra.D, fee); err != nil {
			return err
		}
		if fee.CmpUint64(policyMinFee) < 0 {
			fee.SetUint64(policyMinFee)
		} else if fee.CmpUint64(policyMaxFee) > 0 {
			fee.SetUint64(policyMaxFee)
		}
		return nil
	}
	if t.Extra.PolicyFee != nil { // flat fee override (legacy stored data)
		fee.Set(t.Extra.PolicyFee)
		return nil
	}
	var f uint256.Int
	var err error
	v3 := t.isV3()
	if t.isV21d() || v3 {
		err = feeBalanceTermV21d(xp, t.Extra.FeeGamma, &f)
	} else {
		err = reductionCoefficient(xp, t.Extra.FeeGamma, &f)
	}
	if err != nil {
		return err
	}
	fee.Div(
		number.SafeAdd(
			number.SafeMul(t.Extra.MidFee, &f),
			number.SafeMul(t.Extra.OutFee, number.SafeSub(U_1e18, &f))),
		U_1e18)
	if v3 {
		// A native-fee policy's get_fee is 0, so the pool's own fee applies, within [MIN_FEE, MAX_FEE].
		clampV3Fee(fee)
	}
	return nil
}

// GetDy https://github.com/curvefi/twocrypto-ng/blob/1c800bd/contracts/main/CurveCryptoViews2Optimized.vy#L63
func (t *PoolSimulator) GetDy(
	i int, j int, dx *uint256.Int,

	// output
	dy, fee, K0 *uint256.Int, xp []uint256.Int,
) error {
	// assert dx > 0, "do not exchange 0 coins"
	if dx.IsZero() {
		return ErrExchange0Coins
	}

	A, gamma := t._A_gamma()

	// When A/gamma is actively ramping, recompute D from current balances (mirrors _calc_D_ramp).
	D := t.Extra.D
	ramping := nowUnix() < t.Extra.FutureAGammaTime
	if ramping {
		for k := range NumTokens {
			xp[k].Set(&t.Reserves[k])
		}
		number.SafeMulZ(&xp[0], &t.precisionMultipliers[0], &xp[0])
		for k := range NumTokens - 1 {
			xp[k+1].Div(
				number.SafeMul(number.SafeMul(&xp[k+1], &t.Extra.PriceScale[k]), &t.precisionMultipliers[k+1]),
				Precision,
			)
		}
		var zeroK0 uint256.Int
		var err error
		D, err = newton_D(A, gamma, &zeroK0, xp[:], t.Extra.UseCustomMath)
		if err != nil {
			return err
		}
	}

	yOrg := number.Set(&t.Reserves[j])
	for k := range NumTokens {
		if k == i {
			number.SafeAddZ(&t.Reserves[k], dx, &xp[k])
			continue
		}
		xp[k].Set(&t.Reserves[k])
	}

	number.SafeMulZ(&xp[0], &t.precisionMultipliers[0], &xp[0])
	for k := range NumTokens - 1 {
		xp[k+1].Div(
			number.SafeMul(number.SafeMul(&xp[k+1], &t.Extra.PriceScale[k]), &t.precisionMultipliers[k+1]),
			Precision,
		)
	}

	var y uint256.Int
	var err = get_y(t.Extra.UseCustomMath, j, A, gamma, D, xp[:], &y, K0)
	if err != nil {
		return err
	}

	if y.Cmp(&xp[j]) >= 0 {
		return ErrUnsafeY
	}

	dy.Sub(&xp[j], &y)
	dy.SubUint64(dy, 1)

	xp[j] = y
	if j > 0 {
		dy.Div(number.SafeMul(dy, Precision), &t.Extra.PriceScale[j-1])
	}
	dy.Div(dy, &t.precisionMultipliers[j])

	// During a ramp the pool stores a re-solved D before charging the fee, so the policy's
	// snapshot D no longer matches and it charges the fallback fee.
	err = t.feeCalc(xp[:], fee, !ramping)
	if err != nil {
		return err
	}

	fee.Div(number.SafeMul(fee, dy), U_1e10)
	dy.Sub(dy, fee)

	number.SafeSubZ(yOrg, dy, yOrg)
	number.SafeMulZ(yOrg, &t.precisionMultipliers[j], yOrg)
	if j > 0 {
		yOrg.Div(number.SafeMul(yOrg, &t.Extra.PriceScale[j-1]), Precision)
	}
	xp[j].Set(yOrg)

	return nil
}

// GetDx https://github.com/curvefi/twocrypto-ng/blob/c4093cbda18ec8f3da21bf7e40a3f8d01c5c4bd3/contracts/main/CurveCryptoViews3Optimized.vy#L76
func (t *PoolSimulator) GetDx(
	i int, j int, dy *uint256.Int,

	dx, feeDy, K0 *uint256.Int, xp []uint256.Int,
) error {
	_dy := number.Set(dy)

	for range 5 {
		var err = t._getDxFee(i, j, _dy, dx, K0, xp[:])
		if err != nil {
			return err
		}

		err = t.feeCalc(xp, feeDy, false)
		if err != nil {
			return err
		}
		feeDy.Div(number.SafeMul(feeDy, _dy), U_1e10)
		_dy.Add(dy, number.SafeAdd(feeDy, number.Number_1))
	}

	return nil
}

// https://github.com/curvefi/twocrypto-ng/blob/c4093cbda18ec8f3da21bf7e40a3f8d01c5c4bd3/contracts/main/CurveCryptoViews3Optimized.vy#L184
func (t *PoolSimulator) _getDxFee(
	i int, j int, dy *uint256.Int,

	// output
	dx, K0 *uint256.Int, xp []uint256.Int,
) error {
	// 	assert i != j and i < N_COINS and j < N_COINS, "coin index out of range"
	if i == j || i >= NumTokens || j >= NumTokens {
		return ErrCoinIndexOutOfRange
	}

	// 	assert dy > 0, "do not exchange out 0 coins"
	if dy.Cmp(number.Zero) <= 0 {
		return ErrExchange0Coins
	}

	A, gamma := t._A_gamma()

	// When A/gamma is actively ramping, recompute D from current balances (mirrors _calc_D_ramp).
	D := t.Extra.D
	ramping := nowUnix() < t.Extra.FutureAGammaTime
	if ramping {
		for k := range NumTokens {
			xp[k].Set(&t.Reserves[k])
		}
		number.SafeMulZ(&xp[0], &t.precisionMultipliers[0], &xp[0])
		for k := range NumTokens - 1 {
			xp[k+1].Div(
				number.SafeMul(number.SafeMul(&xp[k+1], &t.Extra.PriceScale[k]), &t.precisionMultipliers[k+1]),
				Precision,
			)
		}
		var zeroK0 uint256.Int
		var err error
		D, err = newton_D(A, gamma, &zeroK0, xp[:], t.Extra.UseCustomMath)
		if err != nil {
			return err
		}
	}

	for k := range NumTokens {
		xp[k].Set(&t.Reserves[k])
	}

	xp[j].Sub(&t.Reserves[j], dy)

	number.SafeMulZ(&xp[0], &t.precisionMultipliers[0], &xp[0])

	for k := range NumTokens - 1 {
		xp[k+1].Div(
			number.SafeMul(number.SafeMul(&xp[k+1], &t.Extra.PriceScale[k]), &t.precisionMultipliers[k+1]),
			Precision,
		)
	}

	var xOut uint256.Int
	err := get_y(t.Extra.UseCustomMath, i, A, gamma, D, xp, &xOut, K0)
	if err != nil {
		return err
	}

	number.SafeSubZ(&xOut, &xp[i], dx)

	xp[i].Set(&xOut)

	if i > 0 {
		dx.Div(number.SafeMul(dx, Precision), &t.Extra.PriceScale[i-1])
	}

	dx.Div(dx, &t.precisionMultipliers[i])

	return nil
}

// https://github.com/curvefi/twocrypto-ng/blob/c4093cbda18ec8f3da21bf7e40a3f8d01c5c4bd3/contracts/main/CurveTwocryptoOptimized.vy#L964
func (t *PoolSimulator) tweak_price(A, gamma *uint256.Int, _xp [NumTokens]uint256.Int, new_D, K0_prev *uint256.Int,
	lastPrices, priceScale, priceOracle []uint256.Int, priceOracleTimestamp *int64,
	xcp_profit, d, virtualPrice *uint256.Int) error {
	/*
				@notice Tweaks price_oracle, last_price and conditionally adjusts
		            price_scale. This is called whenever there is an unbalanced
		            liquidity operation: _exchange, add_liquidity, or
		            remove_liquidity_one_coin.
		    @dev Contains main liquidity rebalancing logic, by tweaking `price_scale`.
		    @param A_gamma Array of A and gamma parameters.
		    @param _xp Array of current balances.
		    @param new_D New D value.
		    @param K0_prev Initial guess for `newton_D`.
	*/

	total_supply := t.Extra.LpSupply
	old_xcp_profit := t.Extra.XcpProfit
	old_virtual_price := t.Extra.VirtualPrice

	blockTimestamp := nowUnix()
	var err error

	// # ----------------------- Update Oracles if needed -----------------------
	// The tracker reads price_oracle() at PriceOracleTimestamp; carry it forward to this block as the
	// contract does once per block. D, virtual_price, xcp_profit, last_prices and price_scale below are
	// updated on every exchange.
	if *priceOracleTimestamp, err = t.priceOracleAt(blockTimestamp, priceOracle); err != nil {
		return err
	}

	// #                  price_oracle is used further on to calculate its vector
	// #            distance from price_scale. This distance is used to calculate
	// #                  the amount of adjustment to be done to the price_scale.

	// # ------------------ If new_D is set to 0, calculate it ------------------
	var D_unadjusted = new_D
	if new_D == nil || new_D.IsZero() {
		D_unadjusted, err = newton_D(A, gamma, K0_prev, _xp[:], t.Extra.UseCustomMath)
		if err != nil {
			return err
		}
	}

	// # ----------------------- Calculate last_prices --------------------------
	err = get_p(t.Extra.UseCustomMath, _xp, D_unadjusted, A, gamma, lastPrices)
	if err != nil {
		return err
	}
	for k := range NumTokens - 1 {
		lastPrices[k].Div(number.SafeMul(&lastPrices[k], &t.Extra.PriceScale[k]), U_1e18)
	}

	// # ---------- Update profit numbers without price adjustment first --------
	var xp [NumTokens]uint256.Int
	xp[0].Div(D_unadjusted, NumTokensU256)
	for k := range NumTokens - 1 {
		xp[k+1].Div(
			number.SafeMul(D_unadjusted, U_1e18),
			number.SafeMul(NumTokensU256, &t.Extra.PriceScale[k]),
		)
	}

	// # ------------------------- Update xcp_profit ----------------------------

	xcp_profit.Set(U_1e18)
	var virtual_price = U_1e18

	if !old_virtual_price.IsZero() {
		xcp := geometric_mean(xp[:])

		virtual_price = number.Div(number.SafeMul(U_1e18, xcp), total_supply)

		xcp_profit.Div(number.SafeMul(old_xcp_profit, virtual_price), old_virtual_price)

		// #       If A and gamma are not undergoing ramps (t < block.timestamp),
		// #         ensure new virtual_price is not less than old virtual_price,
		// #                                        else the pool suffers a loss.
		if t.Extra.FutureAGammaTime < blockTimestamp {
			// assert virtual_price > old_virtual_price, "Loss"
			// v3 only asserts virtual_price >= old (non-strict)
			if c := virtual_price.Cmp(old_virtual_price); c < 0 || (c == 0 && !t.isV3()) {
				return ErrLoss
			}
		}
	}

	// # ------------ Rebalance liquidity if there's enough profits to adjust it:
	// v3 rebalancing (lp_xcp_profit, donations, policy price_scale target) is not modeled: it only
	// moves price_scale after the swap, never the swap's own output.
	if !t.isV3() && number.SafeSub(number.SafeMul(virtual_price, number.Number_2), U_1e18).Cmp(
		number.SafeAdd(xcp_profit, number.SafeMul(number.Number_2, t.Extra.AllowedExtraProfit))) > 0 {
		// # ------------------- Get adjustment step ----------------------------

		// #                Calculate the vector distance between price_scale and
		// #                                                        price_oracle.
		norm := number.Div(number.SafeMul(&priceOracle[0], U_1e18), &t.Extra.PriceScale[0])
		if norm.Cmp(U_1e18) > 0 {
			norm = number.SafeSub(norm, U_1e18)
		} else {
			norm = number.SafeSub(U_1e18, norm)
		}

		var adjustment_step = number.Div(norm, number.Number_5)
		if adjustment_step.Cmp(t.Extra.AdjustmentStep) < 0 {
			adjustment_step.Set(t.Extra.AdjustmentStep)
		}

		if norm.Cmp(adjustment_step) > 0 {
			// # <---------- We only adjust prices if the
			// #          vector distance between price_oracle and price_scale is
			// #             large enough. This check ensures that no rebalancing
			// #           occurs if the distance is low i.e. the pool prices are
			// #                                     pegged to the oracle prices.

			// # ------------------------------------- Calculate new price scale.

			var p_new [NumTokens - 1]uint256.Int
			for k := range NumTokens - 1 {
				p_new[k].Div(
					number.SafeAdd(
						number.SafeMul(&t.Extra.PriceScale[k], number.Sub(norm, adjustment_step)),
						number.SafeMul(adjustment_step, &priceOracle[k]),
					), norm)
			}

			// # ---------------- Update stale xp (using price_scale) with p_new.
			for k := range NumTokens {
				xp[k].Set(&_xp[k])
			}
			for k := range NumTokens - 1 {
				xp[k+1].Div(number.SafeMul(&_xp[k+1], &p_new[k]), &t.Extra.PriceScale[k])
			}

			// # ------------------------------------------ Update D with new xp.
			D, err := newton_D(A, gamma, new(uint256.Int), xp[:], t.Extra.UseCustomMath)
			if err != nil {
				return err
			}

			for k := range NumTokens {
				frac := number.Div(number.SafeMul(&xp[k], U_1e18), D)
				if frac.Cmp(MinFrac) < 0 || frac.Cmp(MaxFrac) > 0 {
					return ErrUnsafeXi
				}
			}

			xp[0].Div(D, NumTokensU256)
			for k := range NumTokens - 1 {
				xp[k+1].Div(number.SafeMul(D, U_1e18), number.SafeMul(NumTokensU256, &p_new[k]))
			}

			// # ---------- Calculate new virtual_price using new xp and D. Reuse
			// #              `old_virtual_price` (but it has new virtual_price).
			temp := geometric_mean(xp[:])
			old_virtual_price = number.Div(number.SafeMul(U_1e18, temp), total_supply)

			// # ---------------------------- Proceed if we've got enough profit.
			if old_virtual_price.Cmp(U_1e18) > 0 && number.SafeSub(number.SafeMul(number.Number_2, old_virtual_price), U_1e18).Cmp(xcp_profit) > 0 {
				for k := range NumTokens - 1 {
					priceScale[k].Set(&p_new[k])
				}
				d.Set(D)
				virtualPrice.Set(old_virtual_price)
				return nil
			}
		}
	}

	// # --------- price_scale was not adjusted. Update the profit counter and D.
	copy(priceScale, t.Extra.PriceScale)
	d.Set(D_unadjusted)
	virtualPrice.Set(virtual_price)
	return nil
}

// priceOracleAt writes price_oracle as the contract computes it at timestamp into out, and returns the
// timestamp it is for. The tracker reads price_oracle() at PriceOracleTimestamp. Until the next trade the
// contract's EMA keeps moving toward last_prices, capped at 2 * price_scale:
//
//	price_oracle(t) = (min(last_prices, 2 * price_scale) * (1e18 - alpha) + price_oracle(t0) * alpha) / 1e18
//	alpha = wad_exp(-(t - t0) * 1e18 / ma_time)
//
// which composes, so carrying the value read at t0 forward by t - t0 gives the contract's value up to
// rounding. Without a read time or ma_time (state from an older tracker), PriceOracle is used as is.
func (t *PoolSimulator) priceOracleAt(timestamp int64, out []uint256.Int) (int64, error) {
	copy(out, t.Extra.PriceOracle)
	readAt := t.Extra.PriceOracleTimestamp
	if readAt == 0 || t.Extra.MaTime == nil || t.Extra.MaTime.IsZero() || timestamp <= readAt {
		return readAt, nil
	}

	var elapsed uint256.Int
	elapsed.SetUint64(uint64(timestamp - readAt))
	exponent := int256.Int(*number.Div(number.SafeMul(&elapsed, U_1e18), t.Extra.MaTime))
	alpha, err := wad_exp(exponent.Neg(&exponent))
	if err != nil {
		return 0, err
	}

	var capped, floor uint256.Int
	v3 := t.isV3()
	for k := range NumTokens - 1 {
		capped.Mul(&t.Extra.PriceScale[k], number.Number_2)
		if t.Extra.LastPrices[k].Lt(&capped) {
			capped.Set(&t.Extra.LastPrices[k])
		}
		// v3.0.0 caps both ways: min(max(last_prices, price_scale / 2), 2 * price_scale).
		if floor.Rsh(&t.Extra.PriceScale[k], 1); v3 && capped.Lt(&floor) {
			capped.Set(&floor)
		}
		out[k].Div(
			number.SafeAdd(
				number.SafeMul(&capped, number.SafeSub(U_1e18, alpha)),
				number.SafeMul(&t.Extra.PriceOracle[k], alpha),
			),
			U_1e18,
		)
	}
	return timestamp, nil
}

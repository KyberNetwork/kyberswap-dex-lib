package tricryptong

import (
	"errors"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

// from contracts/main/CurveTricryptoOptimizedWETH.vy
// (contracts/main/CurveTricryptoOptimized.vy is a bit different and we won't support it for now (there is only 1 pool using that anyway))

func (t *PoolSimulator) FeeCalc(xp []uint256.Int, fee *uint256.Int) error {
	var f uint256.Int
	var err = reductionCoefficient(xp, t.Extra.FeeGamma, &f)
	if err != nil {
		return err
	}
	fee.Div(
		number.SafeAdd(
			number.SafeMul(t.Extra.MidFee, &f),
			number.SafeMul(t.Extra.OutFee, number.SafeSub(U_1e18, &f))),
		U_1e18)
	return nil
}

// GetDy https://github.com/curvefi/tricrypto-ng/blob/c4093cbda18ec8f3da21bf7e40a3f8d01c5c4bd3/contracts/main/CurveCryptoViews3Optimized.vy#L60
func (t *PoolSimulator) GetDy(
	i int, j int, dx *uint256.Int,

	// output
	dy, fee, K0 *uint256.Int, xp []uint256.Int,
) error {
	yOrg := number.Set(&t.Reserves[j])
	for k := 0; k < NumTokens; k += 1 {
		if k == i {
			number.SafeAddZ(&t.Reserves[k], dx, &xp[k])
			continue
		}
		xp[k].Set(&t.Reserves[k])
	}

	number.SafeMulZ(&xp[0], &t.precisionMultipliers[0], &xp[0])
	for k := 0; k < 2; k += 1 {
		xp[k+1].Div(
			number.SafeMul(number.SafeMul(&xp[k+1], &t.Extra.PriceScale[k]), &t.precisionMultipliers[k+1]),
			Precision,
		)
	}

	A, gamma := t._A_gamma()
	var y uint256.Int
	var err = get_y(A, gamma, xp[:], t.Extra.D, j, &y, K0)
	if err != nil {
		return err
	}
	number.SafeSubZ(number.SafeSub(&xp[j], &y), number.Number_1, dy)
	xp[j] = y
	if j > 0 {
		dy.Div(number.SafeMul(dy, U_1e18), &t.Extra.PriceScale[j-1])
	}
	dy.Div(dy, &t.precisionMultipliers[j])

	err = t.FeeCalc(xp[:], fee)
	if err != nil {
		return err
	}

	fee.Div(number.SafeMul(fee, dy), U_1e10)
	dy.Sub(dy, fee)

	number.SafeSubZ(yOrg, dy, yOrg)
	number.SafeMulZ(yOrg, &t.precisionMultipliers[j], yOrg)
	if j > 0 {
		yOrg.Div(number.SafeMul(yOrg, &t.Extra.PriceScale[j-1]), U_1e18)
	}
	xp[j].Set(yOrg)

	return nil
}

// GetDx https://github.com/curvefi/tricrypto-ng/blob/c4093cbda18ec8f3da21bf7e40a3f8d01c5c4bd3/contracts/main/CurveCryptoViews3Optimized.vy#L76
func (t *PoolSimulator) GetDx(
	i int, j int, dy *uint256.Int,

	dx, feeDy, K0 *uint256.Int, xp []uint256.Int,
) error {
	_dy := number.Set(dy)

	for k := 0; k < 5; k += 1 {
		var err = t._getDxFee(i, j, _dy, dx, K0, xp[:])
		if err != nil {
			return err
		}

		err = t.FeeCalc(xp, feeDy)
		if err != nil {
			return err
		}
		feeDy.Div(number.SafeMul(feeDy, _dy), U_1e10)
		_dy.Add(dy, number.SafeAdd(feeDy, number.Number_1))
	}

	return nil
}

// https://github.com/curvefi/tricrypto-ng/blob/c4093cbda18ec8f3da21bf7e40a3f8d01c5c4bd3/contracts/main/CurveCryptoViews3Optimized.vy#L184
func (t *PoolSimulator) _getDxFee(
	i int, j int, dy *uint256.Int,

	// output
	dx, K0 *uint256.Int, xp []uint256.Int,
) error {
	// 	assert i != j and i < N_COINS and j < N_COINS, "coin index out of range"
	if i == j || i >= NumTokens || j >= NumTokens {
		return errors.New("coin index out of range")
	}

	// 	assert dy > 0, "do not exchange out 0 coins"
	if dy.Cmp(number.Zero) <= 0 {
		return errors.New("do not exchange out 0 coins")
	}

	A, gamma := t._A_gamma()
	for k := 0; k < NumTokens; k += 1 {
		xp[k].Set(&t.Reserves[k])
	}

	xp[j].Sub(&t.Reserves[j], dy)

	number.SafeMulZ(&xp[0], &t.precisionMultipliers[0], &xp[0])

	for k := 0; k < 2; k += 1 {
		xp[k+1].Div(
			number.SafeMul(number.SafeMul(&xp[k+1], &t.Extra.PriceScale[k]), &t.precisionMultipliers[k+1]),
			Precision,
		)
	}

	var xOut uint256.Int
	err := get_y(A, gamma, xp, t.Extra.D, i, &xOut, K0)
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

// https://github.com/curvefi/tricrypto-ng/blob/c4093cbda18ec8f3da21bf7e40a3f8d01c5c4bd3/contracts/main/CurveTricryptoOptimizedWETH.vy#L964
func (t *PoolSimulator) tweak_price(A, gamma *uint256.Int, _xp [NumTokens]uint256.Int, new_D, K0_prev *uint256.Int) error {
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
	var last_prices_timestamp = t.Extra.LastPricesTimestamp

	var blockTimestamp = time.Now().Unix()
	var err error

	if last_prices_timestamp < blockTimestamp {
		// this block update price_oracle and last_price_timestamp
		// but in pool tracker we've fetched the calculated price_oracle, not the raw packed value, so we can use that here without updating

		t.Extra.LastPricesTimestamp = blockTimestamp
	}

	// #                  price_oracle is used further on to calculate its vector
	// #            distance from price_scale. This distance is used to calculate
	// #                  the amount of adjustment to be done to the price_scale.

	// # ------------------ If new_D is set to 0, calculate it ------------------
	var D_unadjusted = new_D
	if new_D == nil || new_D.IsZero() {
		D_unadjusted, err = newton_D(A, gamma, _xp[:], K0_prev)
		if err != nil {
			return err
		}
	}

	// # ----------------------- Calculate last_prices --------------------------
	err = get_p(_xp, D_unadjusted, A, gamma, t.Extra.LastPrices)
	if err != nil {
		return err
	}
	for k := 0; k < NumTokens-1; k += 1 {
		t.Extra.LastPrices[k].Div(number.SafeMul(&t.Extra.LastPrices[k], &t.Extra.PriceScale[k]), U_1e18)
	}

	// # ---------- Update profit numbers without price adjustment first --------
	var xp [NumTokens]uint256.Int
	xp[0].Div(D_unadjusted, NumTokensU256)
	for k := 0; k < NumTokens-1; k += 1 {
		xp[k+1].Div(
			number.SafeMul(D_unadjusted, U_1e18),
			number.SafeMul(NumTokensU256, &t.Extra.PriceScale[k]),
		)
	}

	// # ------------------------- Update xcp_profit ----------------------------

	var xcp_profit = U_1e18
	var virtual_price = U_1e18

	if !old_virtual_price.IsZero() {
		xcp := geometric_mean(xp[:])

		virtual_price = number.Div(number.SafeMul(U_1e18, xcp), total_supply)

		xcp_profit = number.Div(number.SafeMul(old_xcp_profit, virtual_price), old_virtual_price)

		// #       If A and gamma are not undergoing ramps (t < block.timestamp),
		// #         ensure new virtual_price is not less than old virtual_price,
		// #                                        else the pool suffers a loss.
		if t.Extra.FutureAGammaTime < blockTimestamp {
			// assert virtual_price > old_virtual_price, "Loss"
			if virtual_price.Cmp(old_virtual_price) <= 0 {
				return errors.New("loss")
			}
		}
	}

	t.Extra.XcpProfit = xcp_profit

	// # ------------ Rebalance liquidity if there's enough profits to adjust it:
	if number.SafeSub(number.SafeMul(virtual_price, number.Number_2), U_1e18).Cmp(
		number.SafeAdd(xcp_profit, number.SafeMul(number.Number_2, t.Extra.AllowedExtraProfit))) > 0 {
		// # ------------------- Get adjustment step ----------------------------

		// #                Calculate the vector distance between price_scale and
		// #                                                        price_oracle.
		var norm = uint256.NewInt(0)
		for k := 0; k < NumTokens-1; k += 1 {
			var ratio = number.Div(number.SafeMul(&t.Extra.PriceOracle[k], U_1e18), &t.Extra.PriceScale[k])
			if ratio.Cmp(U_1e18) > 0 {
				ratio = number.SafeSub(ratio, U_1e18)
			} else {
				ratio = number.SafeSub(U_1e18, ratio)
			}
			norm = number.SafeAdd(norm, number.SafeMul(ratio, ratio))
		}

		norm.Sqrt(norm)

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
			for k := 0; k < NumTokens-1; k += 1 {
				p_new[k].Div(
					number.SafeAdd(
						number.SafeMul(&t.Extra.PriceScale[k], number.Sub(norm, adjustment_step)),
						number.SafeMul(adjustment_step, &t.Extra.PriceOracle[k]),
					), norm)
			}

			// # ---------------- Update stale xp (using price_scale) with p_new.
			for k := 0; k < NumTokens; k += 1 {
				xp[k].Set(&_xp[k])
			}
			for k := 0; k < NumTokens-1; k += 1 {
				xp[k+1].Div(number.SafeMul(&_xp[k+1], &p_new[k]), &t.Extra.PriceScale[k])
			}

			// # ------------------------------------------ Update D with new xp.
			D, err := newton_D(A, gamma, xp[:], uint256.NewInt(0))
			if err != nil {
				return err
			}

			for k := 0; k < NumTokens; k += 1 {
				frac := number.Div(number.SafeMul(&xp[k], U_1e18), D)
				if frac.Cmp(MinFrac) < 0 || frac.Cmp(MaxFrac) > 0 {
					return errors.New("unsafe values x[i]")
				}
			}

			xp[0].Div(D, NumTokensU256)
			for k := 0; k < NumTokens-1; k += 1 {
				xp[k+1].Div(number.SafeMul(D, U_1e18), number.SafeMul(NumTokensU256, &p_new[k]))
			}

			// # ---------- Calculate new virtual_price using new xp and D. Reuse
			// #              `old_virtual_price` (but it has new virtual_price).
			temp := geometric_mean(xp[:])
			old_virtual_price = number.Div(number.SafeMul(U_1e18, temp), total_supply)

			// # ---------------------------- Proceed if we've got enough profit.
			if old_virtual_price.Cmp(U_1e18) > 0 && number.SafeSub(number.SafeMul(number.Number_2, old_virtual_price), U_1e18).Cmp(xcp_profit) > 0 {
				for k := 0; k < NumTokens-1; k += 1 {
					t.Extra.PriceScale[k].Set(&p_new[k])
				}
				t.Extra.D.Set(D)
				t.Extra.VirtualPrice.Set(old_virtual_price)
				return nil
			}
		}
	}

	// # --------- price_scale was not adjusted. Update the profit counter and D.
	t.Extra.D.Set(D_unadjusted)
	t.Extra.VirtualPrice.Set(virtual_price)
	return nil
}

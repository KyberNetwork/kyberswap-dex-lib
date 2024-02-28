package ambient

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient/tickmath"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient/types"
)

/* All CrocSwap swaps occur as legs across locally stable constant-product AMM
* curves. For large moves across tick boundaries, the state of this curve might
* change as range-bound liquidity is kicked in or out of the currently active
* curve. But for small moves within tick boundaries (or between tick boundaries
* with no liquidity bumps), the curve behaves like a classic constant-product AMM.
*
* CrocSwap tracks two types of liquidity. 1) Ambient liquidity that is non-
* range bound and remains active at all prices from zero to infinity, until
* removed by the staking user. 2) Concentrated liquidity that is tied to an
* arbitrary lower<->upper tick range and is kicked out of the curve when the
* price moves out of range.
*
* In the CrocSwap model all collected fees are directly incorporated as expanded
* liquidity onto the curve itself. (See CurveAssimilate.sol for more on the
* mechanics.) All accumulated fees are added as ambient-type liquidity, even those
* fees that belong to the pro-rata share of the active concentrated liquidity.
* This is because on an aggregate level, we can't break down the pro-rata share
* of concentrated rewards to the potentially near infinite concentrated range
* possibilities.
*
* Because of this concentrated liquidity can be flatly represented as 1:1 with
* contributed liquidity. Ambient liquidity, in contrast, deflates over time as
* it accumulates rewards. Therefore it's represented in terms of seed amount,
* i.e. the equivalent of 1 unit of ambient liquidity contributed at the inception
* of the pool. As fees accumulate the conversion rate from seed to liquidity
* continues to increase.
*
* Finally concentrated liquidity rewards are represented in terms of accumulated
* ambient seeds. This automatically takes care of the compounding of ambient
* rewards compounded on top of concentrated rewards.
*
* @param priceRoot_ The square root of the price ratio exchange rate between the
*   base and quote-side tokens in the AMM curve. (represented in Q64.64 fixed point)
* @param ambientSeeds_ The total ambient liquidity seeds in the current curve.
*   (Inflated by seed deflator to get efective ambient liquidity)
* @param concLiq_ The total concentrated liquidity active and in range at the
*   current state of the curve.
* @param seedDeflator_ The cumulative growth rate (represented as Q16.48 fixed
*    point) of a hypothetical 1-unit of ambient liquidity held in the pool since
*    inception.
* @param concGrowth_ The cumulative rewards growth rate (represented as Q16.48
*   fixed point) of hypothetical 1 unit of concentrated liquidity in range in the
*   pool since inception.
*
* @dev Price ratio is stored as a square root because it makes reserve calculation
*      arithmetic much easier. To be conservative with collateral these growth
*      rates should always be rounded down from their real-value results. Some
*      minor lower-bound approximation is fine, since all it will result in is
*      slightly smaller reward payouts. */
type curveState struct {
	priceRoot    *big.Int
	ambientSeeds *big.Int
	concLiq      *big.Int
	seedDeflator uint64
	concGrowth   uint64
}

/*
	@notice Applies the swap on to the liquidity curve, either fully exhausting

*   the swap or reaching the concentrated liquidity bounds or the user-specified
*   limit price. After calling, the curve and swap objects will be updated with
*   the swap price impact, the liquidity fees assimilated into the curve's ambient
*   liquidity, and the swap accumulators incremented with the cumulative flows.
*
* @param curve - The current in-range liquidity curve. After calling, price and
*    fee accumulation will be adjusted based on the swap processed in this leg.
* @param accum - An accumulator for the asset pair the swap/curve applies to.
*    This object will be incremented with the flow processed on this leg. The swap
*    may or may not be fully exhausted. Caller should check the swap.qty_ field.
@ @param swap - The user directive specifying the swap to execute on this curve.
*    Defines the direction, size, and limit price. After calling, the swapQty will
*    be decremented with the amount of size executed in this leg.
* @param pool - The specifications for the pool's AMM curve, notably in this context
*    the fee rate and protocol take.     *
* @param bumpTick - The tick boundary, past which the constant product AMM
*    liquidity curve is no longer known to be valid. (Either because it represents
*    a liquidity bump point, or the end of a tick bitmap horizon.) The curve will
*    never move past this tick boundary in the call. Caller's responsibility is to
*    set this parameter in the correct direction. I.e. buys should be the boundary
*    from above and sells from below. Represented as a price tick index.
*/
func (c *curveState) swapToLimit(accum *pairFlow, swap *swapDirective, p swapPool, bumpTick types.Int24) error {
	// uint128 limitPrice = determineLimit(bumpTick, swap.limitPrice_, swap.isBuy_);
	limitPrice, err := c.determineLimit(bumpTick, swap.limitPrice, swap.isBuy)
	if err != nil {
		return err
	}

	// (int128 paidBase, int128 paidQuote, uint128 paidProto) =
	// bookExchFees(curve, swap.qty_, pool, swap.inBaseQty_, limitPrice);
	paidBase, paidQuote, paidProto, err := c.bookExchFees(swap.qty, p, swap.inBaseQty, limitPrice)
	if err != nil {
		return err
	}

	// accum.accumSwap(swap.inBaseQty_, paidBase, paidQuote, paidProto);
	accum.accumSwap(swap.inBaseQty, paidBase, paidQuote, paidProto)

	return nil
}

/* @notice Calculates exchange fee charge based off an estimate of the predicted
*         order flow on this leg of the swap.
*
* @dev    Note that the process of collecting the exchange fee itself alters the
*   structure of the curve, because those fees assimilate as liquidity into the
*   curve new liquidity. As such the flow used to pro-rate fees is only an estimate
*   of the actual flow that winds up executed. This means that fees are not exact
*   relative to realized flows. But because fees only have a small impact on the
*   curve, they'll tend to be very close. Getting fee exactly correct doesn't
*   matter, and either over or undershooting is fine from a collateral stability
*   perspective. */
func (c *curveState) bookExchFees(
	swapQty *big.Int, p swapPool, inBaseQty bool, limitPrice *big.Int,
) (*big.Int, *big.Int, *big.Int, error) {
	// (uint128 liqFees, uint128 exchFees) = calcFeeOverSwap
	// (curve, swapQty, pool.feeRate_, pool.protocolTake_, inBaseQty, limitPrice);
	liqFees, exchFees, err := c.calcFeeOverSwap(swapQty, p.feeRate, p.protocolTake, inBaseQty, limitPrice)
	if err != nil {
		return nil, nil, nil, err
	}

	// /* We can guarantee that the price shift associated with the liquidity
	// * assimilation is safe. The limit price boundary is by definition within the
	// * tick price boundary of the locally stable AMM curve (see determineLimit()
	// * function). The liquidity assimilation flow is mathematically capped within
	// * the limit price flow, because liquidity fees are a small fraction of swap
	// * flows. */
	// curve.assimilateLiq(liqFees, inBaseQty);
	c.assimilateLiq(liqFees, inBaseQty)

	// return assignFees(liqFees, exchFees, inBaseQty);
	paidBase, paidQuote, paidProto := assignFees(liqFees, exchFees, inBaseQty)
	return paidBase, paidQuote, paidProto, nil
}

/* @notice Determines an effective limit price given the combination of swap-
*    specified limit, tick liquidity bump boundary on the locally stable AMM curve,
*    and the numerical boundaries of the price field. Always picks the value that's
*    most to the inside of the swap direction. */
var (
	big1 = big.NewInt(1)
)

func (c curveState) determineLimit(bumpTick types.Int24, limitPrice *big.Int, isBuy bool) (*big.Int, error) {
	// uint128 bounded = boundLimit(bumpTick, limitPrice, isBuy);
	bounded, err := c.boundLimit(bumpTick, limitPrice, isBuy)
	if err != nil {
		return nil, err
	}

	// if (bounded < TickMath.MIN_SQRT_RATIO)  return TickMath.MIN_SQRT_RATIO;
	if bounded.Cmp(tickmath.MIN_SQRT_RATIO) == -1 {
		return new(big.Int).Set(tickmath.MIN_SQRT_RATIO), nil
	}

	// if (bounded >= TickMath.MAX_SQRT_RATIO) return TickMath.MAX_SQRT_RATIO - 1; // Well above 0, cannot underflow
	if bounded.Cmp(tickmath.MAX_SQRT_RATIO) > -1 {
		r := new(big.Int).Set(tickmath.MAX_SQRT_RATIO)
		r.Sub(r, big1)
		return r, nil
	}

	return bounded, nil
}

/* @notice Finds the effective max (min) swap limit price giving a bump tick index
*         boundary and a user specified limitPrice.
*
* @dev Because the mapping from ticks to bumps always occur at the lowest price unit
*      inside a tick, there is an asymmetry between the lower and upper bump tick arg.
*      The lower bump tick is the lowest tick *inclusive* for which liquidity is active.
*      The upper bump tick is the *next* tick above where liquidity is active. Therefore
*      the lower liquidity price maps to the bump tick price, whereas the upper liquidity
*      price bound maps to one unit less than the bump tick price.
*
*     Lower bump price                             Upper bump price
*            |                                           |
*      ------X******************************************+X-----------------
*            |                                          |
*     Min liquidity prce                         Max liquidity price
 */
var (
	TICK_STEP_SHAVE_DOWN = big.NewInt(1)
)

func (c curveState) boundLimit(bumpTick types.Int24, limitPrice *big.Int, isBuy bool) (*big.Int, error) {
	// if (bumpTick <= TickMath.MIN_TICK || bumpTick >= TickMath.MAX_TICK) {
	// 	   return limitPrice;
	//  }
	if bumpTick <= tickmath.MIN_TICK || bumpTick >= tickmath.MAX_TICK {
		return limitPrice, nil
	}

	// if (isBuy) {
	// 	/* See comment above. Upper bound liquidity is last active at the price one unit
	// 	 * below the upper tick price. */
	// 	uint128 TICK_STEP_SHAVE_DOWN = 1;
	// 	// Valid uint128 root prices are always well above 0.
	// 	uint128 bumpPrice = TickMath.getSqrtRatioAtTick(bumpTick) - TICK_STEP_SHAVE_DOWN;
	// 	return bumpPrice < limitPrice ? bumpPrice : limitPrice;
	// }

	if isBuy {
		bumpPrice, err := tickmath.GetSqrtRatioAtTick(bumpTick)
		if err != nil {
			return nil, err
		}
		bumpPrice.Sub(bumpPrice, TICK_STEP_SHAVE_DOWN)
		if bumpPrice.Cmp(limitPrice) == -1 {
			return bumpPrice, nil
		}
		return limitPrice, nil
	}

	// uint128 bumpPrice = TickMath.getSqrtRatioAtTick(bumpTick);
	// return bumpPrice > limitPrice ? bumpPrice : limitPrice;
	bumpPrice, err := tickmath.GetSqrtRatioAtTick(bumpTick)
	if err != nil {
		return nil, err
	}
	if bumpPrice.Cmp(limitPrice) == -1 {
		return bumpPrice, nil
	}
	return limitPrice, nil
}

/* @notice Calculates the exchange fee given a swap directive and limitPrice. Note
*   this assumes the curve is constant-product without liquidity bumps through the
*   whole range. Don't use this function if you're unable to guarantee that the AMM
*   curve is locally stable through the price impact.
*
* @param curve The current state of the AMM liquidity curve. Must be stable without
*              liquidity bumps through the price impact.
* @param swapQty The quantity specified for this leg of the swap, may or may not be
*                fully executed depending on limitPrice.
* @param feeRate The pool's fee as a proportion of notion executed. Represented as
*                a multiple of 0.0001%
* @param protoTake The protocol's take as a share of the exchange fee. (Rest goes to
*                  liquidity rewards.) Represented as 1/n (with zero a special case.)
* @param inBaseQty If true the swap quantity is denominated as base-side tokens. If
*                  false, quote-side tokens.
* @param limitPrice The max (min) price this leg will swap to if it's a buy (sell).
*                   Represented as the square root of price as a Q64.64 fixed-point.
*
* @return liqFee The total fees that's allocated as liquidity rewards accumulated
*                to liquidity providers in the pool (in the opposite side tokens of
*                the swap denomination).
* @return protoFee The total fee accumulated as CrocSwap protocol fees. */
func (c curveState) calcFeeOverSwap(
	swapQty *big.Int, feeRate uint16, protoTake uint8, inBaseQty bool, limitPrice *big.Int,
) (*big.Int, *big.Int, error) {
	// uint128 flow = curve.calcLimitCounter(swapQty, inBaseQty, limitPrice);
	flow, err := c.calcLimitCounter(swapQty, inBaseQty, limitPrice)
	if err != nil {
		return nil, nil, err
	}

	// (liqFee, protoFee) = calcFeeOverFlow(flow, feeRate, protoTake);
	liqFee, protoFee := calcFeeOverFlow(flow, feeRate, protoTake)

	return liqFee, protoFee, nil
}

/* @notice Given a fixed flow and a fee rate, calculates the owed liquidty and
*         protocol fees. */
var (
	// uint256 FEE_BP_MULT = 1_000_000;
	FEE_BP_MULT = big.NewInt(1_000_000)
	big256      = big.NewInt(256)
)

func calcFeeOverFlow(flow *big.Int, feeRate uint16, protoProp uint8) (*big.Int, *big.Int) {
	// 	   // Guaranteed to fit in 256 bit arithmetic. Safe to cast back to uint128
	// 	   // because fees will never be larger than the underlying flow.
	// 	   uint256 totalFee = (uint256(flow) * feeRate) / FEE_BP_MULT;
	totalFee := new(big.Int).Mul(flow, big.NewInt(int64(feeRate)))
	totalFee.Div(totalFee, FEE_BP_MULT)

	// protoFee = uint128(totalFee * protoProp / 256);
	protoFee := new(big.Int).Mul(totalFee, big.NewInt(int64(protoProp)))
	protoFee.Div(protoFee, big256)

	// liqFee = uint128(totalFee) - protoFee;
	liqFee := new(big.Int).Sub(totalFee, protoFee)

	return protoFee, liqFee
}

// function calcFeeOverFlow (uint128 flow, uint16 feeRate, uint8 protoProp)
//    private pure returns (uint128 liqFee, uint128 protoFee) {
//    unchecked {
//

//    }
// }

/* @notice Similar to calcLimitFlows(), except returns the max possible flow in the
*   *opposite* direction. I.e. if inBaseQty_ is True, returns the quote token flow
*   for the swap. And vice versa..
*
* @dev The fixed-point result approximates the real valued formula with close but
*   directionally unpredicable precision. It could be slightly above or slightly
*   below. In the case of zero flows this could be substantially over. This
*   function should not be used in any context with strict directional boundness
*   requirements. */
func (c curveState) calcLimitCounter(swapQty *big.Int, inBaseQty bool, limitPrice *big.Int) (*big.Int, error) {
	// bool isBuy = limitPrice > curve.priceRoot_;
	isBuy := limitPrice.Cmp(c.priceRoot) == 1

	// uint128 denomFlow = calcLimitFlows(curve, swapQty, inBaseQty, limitPrice);
	denomFlow, err := c.calcLimitFlow(swapQty, inBaseQty, limitPrice)
	if err != nil {
		return nil, err
	}

	// return invertFlow(activeLiquidity(curve), curve.priceRoot_,
	//	   denomFlow, isBuy, inBaseQty);
	activeLiq, err := c.activeLiquidity()
	if err != nil {
		return nil, err
	}
	res := invertFlow(activeLiq, c.priceRoot, denomFlow, isBuy, inBaseQty)

	return res, nil
}

/* @dev The fixed point arithmetic results in output that's a close approximation
*   to the true real value, but could be skewed in either direction. The output
*   from this function should not be consumed in any context that requires strict
*   boundness. */
var (
	big0 = big.NewInt(0)
)

func invertFlow(liq *big.Int, price *big.Int, denowFlow *big.Int, isBuy bool, inBaseQty bool) *big.Int {
	// if (liq == 0) { return 0; }
	if liq.Cmp(big0) == 0 {
		return big.NewInt(0)
	}

	// uint256 invertReserve = reserveAtPrice(liq, price, !inBaseQty);
	// uint256 initReserve = reserveAtPrice(liq, price, inBaseQty);
	invertReserve := reserveAtPrice(liq, price, !inBaseQty)
	initReserve := reserveAtPrice(liq, price, inBaseQty)

	// uint256 endReserve = (isBuy == inBaseQty) ?
	// initReserve + denomFlow : // Will always fit in 256-bits
	// initReserve - denomFlow; // flow is always less than total reserves
	endReserve := new(big.Int)
	if isBuy == inBaseQty {
		endReserve.Add(initReserve, denowFlow)
	} else {
		endReserve.Sub(initReserve, denowFlow)
	}

	// if (endReserve == 0) { return type(uint128).max; }
	if endReserve.Cmp(big0) == 0 {
		return new(big.Int).Set(bigMaxUint128)
	}

	// uint256 endInvert = uint256(liq) * uint256(liq) / endReserve;
	endInvert := new(big.Int).Mul(liq, liq)
	endInvert.Div(endInvert, endReserve)

	// return (endInvert > invertReserve ?
	// endInvert - invertReserve :
	// invertReserve - endInvert).toUint128();
	if endInvert.Cmp(invertReserve) == 1 {
		return endInvert.Sub(endInvert, invertReserve)
	}

	return invertReserve.Sub(invertReserve, endInvert)
}

/* @notice Calculates the total quantity of tokens that can be swapped on the AMM
*   curve until either 1) the limit price is reached or 2) the swap fills its
*   entire remaining quantity.
*
* @dev This function does *NOT* account for the possibility of concentrated liq
*   being knocked in/out as the price on the AMM curve moves across tick boundaries.
*   It's the responsibility of the caller to properly check whether the limit price
*   is within the bounds of the locally stable curve.
*
* @dev As long as CurveState's fee accum fields are conservatively lower bounded,
*   and as long as limitPrice is accurate, then this function rounds down from the
*   true real value. At most this round down loss of precision is tightly bounded at
*   2 wei. (See comments in deltaPriceQuote() function)
*
* @param curve - The current state of the liquidity curve. No guarantee that it's
*   liquidity stable through the entire limit range (see @dev above). Note that this
*   function does *not* update the curve struct object.
* @param swapQty - The total remaining quantity left in the swap.
* @param inBaseQty - Whether the swap quantity is denomianted in base or quote side
*                    token.
* @param limitPrice - The highest (lowest) acceptable ending price of the AMM curve
*   for a buy (sell) swap. Represented as Q64.64 fixed point square root of the
*   price.
*
* @return - The maximum executable swap flow (rounded down by fixed precision).
*           Denominated on the token side based on inBaseQty param. Will
*           always return unsigned magnitude regardless of the direction. User
*           can easily determine based on swap context. */

func (c curveState) calcLimitFlow(swapQty *big.Int, inBaseQty bool, limitPrice *big.Int) (*big.Int, error) {
	// uint128 limitFlow = calcLimitFlows(curve, inBaseQty, limitPrice);
	limitFlow, err := c.calcLimitFlows2(inBaseQty, limitPrice)
	if err != nil {
		return nil, err
	}

	// return limitFlow > swapQty ? swapQty : limitFlow;
	if limitFlow.Cmp(swapQty) == 1 {
		return swapQty, nil
	}
	return limitFlow, nil
}

func (c curveState) calcLimitFlows2(inBaseQty bool, limitPrice *big.Int) (*big.Int, error) {
	// uint128 liq = activeLiquidity(curve);
	liq, err := c.activeLiquidity()
	if err != nil {
		return nil, err
	}

	// return inBaseQty ?
	// deltaBase(liq, curve.priceRoot_, limitPrice) :
	// deltaQuote(liq, curve.priceRoot_, limitPrice);
	// }
	if inBaseQty {
		return deltaBase(liq, c.priceRoot, limitPrice), nil
	}
	return deltaQuote(liq, c.priceRoot, limitPrice), nil
}

/* @notice Calculates the total amount of liquidity represented by the liquidity
*         curve object.
* @dev    Result always rounds down from the real value, *assuming* that the fee
*         accumulation fields are conservative lower-bound rounded.
* @param curve - The currently active liqudity curve state. Remember this curve
*    state is only known to be valid within the current tick.
* @return - The total scalar liquidity. Equivalent to sqrt(X*Y) in an equivalent
*           constant-product AMM. */
func (c curveState) activeLiquidity() (*big.Int, error) {
	// uint128 ambient = CompoundMath.inflateLiqSeed(curve.ambientSeeds_, curve.seedDeflator_);
	ambient := inflateLiqSeed(c.ambientSeeds, c.seedDeflator)

	// return LiquidityMath.addLiq(ambient, curve.concLiq_);
	return addLiq(ambient, c.concLiq)
}

/* @notice Returns the amount of virtual reserves give the price and liquidity of the
*   constant-product liquidity curve.
*
* @dev The actual pool probably holds significantly less collateral because of the
*   use of concentrated liquidity.
* @dev Results always round down from the precise real-valued mathematical result.
*
* @param liq - The total active liquidity in AMM curve. Represented as sqrt(X*Y)
* @param price - The current active (square root of) price of the AMM curve.
*                 represnted as Q64.64 fixed point
* @param inBaseQty - The side of the pool to calculate the virtual reserves for.
*
* @returns The virtual reserves of the token (rounded down to nearest integer).
*   Equivalent to the amount of tokens that would be held for an equivalent
*   classical constant- product AMM without concentrated liquidity.  */
func reserveAtPrice(liq *big.Int, price *big.Int, inBaseQty bool) *big.Int {
	// return (inBaseQty ?
	// uint256(FixedPoint.mulQ64(liq, price)) :
	// uint256(FixedPoint.divQ64(liq, price))).toUint128();

	if inBaseQty {
		return mulQ64(liq, price)
	}

	return divQ64(liq, price)
}

// function reserveAtPrice (uint128 liq, uint128 price, bool inBaseQty)
// internal pure returns (uint128) {

// }

/* @notice Calculates the change to base token reserves associated with a price
*   move along an AMM curve of constant liquidity.
*
* @dev Result is a tight lower-bound for fixed-point precision. Meaning if the
*   the returned limit is X, then X will be inside the limit price and (X+1)
*   will be outside the limit price. */
// function deltaBase (uint128 liq, uint128 priceX, uint128 priceY)
// 	internal pure returns (uint128) {
// 	unchecked {
// Condition assures never underflows
//
// 	}
// }
func deltaBase(liq *big.Int, priceX *big.Int, priceY *big.Int) *big.Int {
	// 	uint128 priceDelta = priceX > priceY ?
	// 		priceX - priceY : priceY - priceX;
	var priceDelta *big.Int
	if priceX.Cmp(priceY) == 1 {
		priceDelta = new(big.Int).Sub(priceX, priceY)
		return priceDelta
	}
	priceDelta = new(big.Int).Sub(priceY, priceX)

	// return reserveAtPrice(liq, priceDelta, true);
	return reserveAtPrice(liq, priceDelta, true)
}

/* @notice Calculates the change to quote token reserves associated with a price
*   move along an AMM curve of constant liquidity.
*
* @dev Result is almost always within a fixed-point precision unit from the true
*   real value. However in certain rare cases, the result could be up to 2 wei
*   below the true mathematical value. Caller should account for this */
func deltaQuote(liq *big.Int, price *big.Int, limitPrice *big.Int) *big.Int {
	// 	function deltaQuote (uint128 liq, uint128 price, uint128 limitPrice)
	// internal pure returns (uint128) {
	// // For purposes of downstream calculations, we make sure that limit price is
	// // larger. End result is symmetrical anyway
	// if (limitPrice > price) {
	// 	return calcQuoteDelta(liq, limitPrice, price);
	// } else {
	// 	return calcQuoteDelta(liq, price, limitPrice);
	// }
	// }

	if limitPrice.Cmp(price) == 1 {
		return calcQuoteDelta(liq, limitPrice, price)
	}

	return calcQuoteDelta(liq, price, limitPrice)
}

/* The formula calculated is
*    F = L * d / (P*P')
*   (where F is the flow to the limit price, where L is liquidity, d is delta,
*    P is price and P' is limit price)
*
* Calculating this requires two stacked mulDiv. To meet the function's contract
* we need to compute the result with tight fixed point boundaries at or below
* 2 wei to conform to the function's contract.
*
* The fixed point calculation of flow is
*    F = mulDiv(mulDiv(...)) = FR - FF
*  (where F is the fixed point result of the formula, FR is the true real valued
*   result with inifnite precision, FF is the loss of precision fractional round
*   down, mulDiv(...) is a fixed point mulDiv call of the form X*Y/Z)
*
* The individual fixed point terms are
*    T1 = mulDiv(X1, Y1, Z1) = T1R - T1F
*    T2 = mulDiv(T1, Y2, Z2) = T2R - T2F
*  (where T1 and T2 are the fixed point results from the first and second term,
*   T1R and T2R are the real valued results from an infinite precision mulDiv,
*   T1F and T2F are the fractional round downs, X1/Y1/Z1/Y2/Z2 are the arbitrary
*   input terms in the fixed point calculation)
*
* Therefore the total loss of precision is
*    FF = T2F + T1F * T2R/T1
*
* To guarantee a 2 wei precision loss boundary:
*    FF <= 2
*    T2F + T1F * T2R/T1 <= 2
*    T1F * T2R/T1 <=  1      (since T2F as a round-down is always < 1)
*    T2R/T1 <= 1             (since T1F as a round-down is always < 1)
*    Y2/Z2 >= 1
*    Z2 >= Y2 */
func calcQuoteDelta(liq *big.Int, priceBig *big.Int, priceSmall *big.Int) *big.Int {
	// uint128 priceDelta = priceBig - priceSmall;
	priceDelta := new(big.Int).Sub(priceBig, priceSmall)

	// // This is cast to uint256 but is guaranteed to be less than 2^192 based off
	// // the return type of divQ64
	// uint256 termOne = FixedPoint.divQ64(liq, priceSmall);
	termOne := divQ64(liq, priceSmall)

	// // As long as the final result doesn't overflow from 128-bits, this term is
	// // guaranteed not to overflow from 256 bits. That's because the final divisor
	// // can be at most 128-bits, therefore this intermediate term must be 256 bits
	// // or less.
	// //
	// // By definition priceBig is always larger than priceDelta. Therefore the above
	// // condition of Z2 >= Y2 is satisfied and the equation caps at a maximum of 2
	// // wei of precision loss.
	// uint256 termTwo = termOne * uint256(priceDelta) / uint256(priceBig);
	// return termTwo.toUint128();
	termTwo := termOne.Mul(termOne, priceDelta)
	termTwo = termTwo.Div(termTwo, priceBig)

	return termTwo
}

/* @notice Computes the amount of token over-collateralization needed to buffer any
*   loss of precision rounding in the fixed price arithmetic on curve price. This
*   is necessary because price occurs in different units than tokens, and we can't
*   assume a single wei is sufficient to buffer one price unit.
*
* @dev In practice the price unit precision is almost always smaller than the token
*   token precision. Therefore the result is usually just 1 wei. The exception are
*   pools where liquidity is very high or price is very low.
*
* @param liq The total liquidity in the curve.
* @param price The (square root) price of the curve in Q64.64 fixed point
* @param inBase If true calculate the token precision on the base side of the pair.
*               Otherwise, calculate on the quote token side.
*
* @return The conservative upper bound in number of tokens that should be
*   burned to over-collateralize a single precision unit of price rounding. If
*   the price arithmetic involves multiple units of precision loss, this number
*   should be multiplied by that factor. */
func priceToTokenPrecision(liq *big.Int, price *big.Int, inBase bool) *big.Int {
	// To provide more base token collateral than price precision rounding:
	//     delta(B) >= L * delta(P)
	//     delta(P) <= 2^-64  (64 bit precision rounding)
	//     delta(B) >= L * 2^-64
	//  (where L is liquidity, B is base token reserves, P is price)
	// if (inBase) {
	// // Since liq is shifted right by 64 bits, adding one can never overflow
	// return (liq >> 64) + 1;
	if inBase {
		res := new(big.Int).Set(liq)
		res.Rsh(res, 64)
		res.Add(res, big1)
		return res
	}

	// Calculate the quote reservs at the current price and a one unit price step,
	// then take the difference as the minimum required quote tokens needed to
	// buffer that price step.
	// uint192 step = FixedPoint.divQ64(liq, price - 1);
	tmpPrice := new(big.Int).Set(price)
	tmpPrice.Sub(tmpPrice, big1)
	step := divQ64(liq, tmpPrice)

	// uint192 start = FixedPoint.divQ64(liq, price);
	start := divQ64(liq, price)

	// next reserves will always be equal or greater than start reserves, so the
	// subtraction will never underflow.
	// uint192 delta = step - start;
	delta := new(big.Int).Sub(step, start)

	// Round tokens up conservative.
	// This will never overflow because 192 bit nums incremented by 1 will always fit in
	// 256 bits.
	// uint256 deltaRound = uint256(delta) + 1;
	deltaRound := delta.Add(delta, big1)

	// return deltaRound.toUint128();
	return deltaRound
}

// 	 function priceToTokenPrecision (uint128 liq, uint128 price,
// 		bool inBase) internal pure returns (uint128) {
// unchecked {

// } else {
//
// }
// }
// }

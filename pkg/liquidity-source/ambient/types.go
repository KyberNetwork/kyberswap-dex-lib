package ambient

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient/tickmath"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient/types"
)

const (
	DexType    = "ambient"
	fetchLimit = 1000
)

type FetchPoolsResponse struct {
	Pools []Pool `json:"pools"`
}

type Pool struct {
	ID          string `json:"id"`
	BlockCreate string `json:"blockCreate"`
	TimeCreate  uint64 `json:"timeCreate,string"`
	Base        string `json:"base"`
	Quote       string `json:"quote"`
	PoolIdx     string `json:"poolIdx"`
}

type PoolListUpdaterMetadata struct {
	LastCreateTime uint64 `json:"lastCreateTime"`
}

type StaticExtra struct {
	Base    string `json:"base"`
	Quote   string `json:"quote"`
	PoolIdx string `json:"pool_idx"`
}

type PoolData struct {
}

type swapPool struct {
	hash string
}

/* @notice Represents the accumulated flow between user and pool within a transaction.
*
* @param baseFlow_ Represents the cumulative base side token flow. Negative for
*   flow going to the user, positive for flow going to the pool.
* @param quoteFlow_ The cumulative quote side token flow.
* @param baseProto_ The total amount of base side tokens being collected as protocol
*   fees. The above baseFlow_ value is inclusive of this quantity.
* @param quoteProto_ The total amount of quote tokens being collected as protocol
*   fees. The above quoteFlow_ value is inclusive of this quantity. */
type pairFlow struct {
	baseFlow   *big.Int
	quoteFlow  *big.Int
	baseProto  *big.Int
	quoteProto *big.Int
}

/* @notice Defines a single requested swap on a pre-specified pool.
*
* @dev A directive indicating no swap action must set *both* qty and limitPrice to
*      zero. qty=0 alone will indicate the use of a flexible back-filled rolling
*      quantity.
*
* @param isBuy_ If true, swap converts base-side token to quote-side token.
*               Vice-versa if false.
* @param inBaseQty_ If true, swap quantity is denominated in base-side token.
*                   If false in quote side token.
* @param rollType_  The flavor of rolling gap fill that should be applied (if any)
*                   to this leg of the directive. See Chaining.sol for list of
*                   rolling type codes.
* @param qty_ The total amount to be swapped. (Or rolling target if rollType_ is
*             enabled)
* @param limitPrice_ The maximum (minimum) *price to pay, if a buy (sell) swap
*           *at the margin*. I.e. the swap will keep exeucting until the curve
*           reaches this price (or exhausts the specified quantity.) Represented
*           as the square root of the pool's price ratio in Q64.64 fixed-point. */
type swapDirective struct {
	isBuy      bool
	inBaseQty  bool
	rollType   uint8
	qty        *big.Int
	limitPrice *big.Int
}

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
func (c curveState) swapToLimit(accum pairFlow, swap swapDirective, p swapPool, bumpTick types.Int24) error {
	limitPrice, err := c.determineLimit(bumpTick, swap.limitPrice, swap.isBuy)
	if err != nil {
		return err
	}

	return nil
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
func (c curveState) bookExchFees(
	swapQty *big.Int, p swapPool, inBaseQty bool, limitPrice *big.Int,
) (*big.Int, *big.Int, *big.Int, error) {
	// 	(uint128 liqFees, uint128 exchFees) = calcFeeOverSwap
	// (curve, swapQty, pool.feeRate_, pool.protocolTake_, inBaseQty, limitPrice);

	// /* We can guarantee that the price shift associated with the liquidity
	// * assimilation is safe. The limit price boundary is by definition within the
	// * tick price boundary of the locally stable AMM curve (see determineLimit()
	// * function). The liquidity assimilation flow is mathematically capped within
	// * the limit price flow, because liquidity fees are a small fraction of swap
	// * flows. */
	// curve.assimilateLiq(liqFees, inBaseQty);

	// return assignFees(liqFees, exchFees, inBaseQty);

	return nil, nil, nil, nil
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
	swapQty *big.Int, feeRate *big.Int, protoTake uint8, inBaseQty bool, limitPrice *big.Int,
) (*big.Int, *big.Int) {
	// uint128 flow = curve.calcLimitCounter(swapQty, inBaseQty, limitPrice);
	// (liqFee, protoFee) = calcFeeOverFlow(flow, feeRate, protoTake);
	return nil, nil
}

/* @notice Similar to calcLimitFlows(), except returns the max possible flow in the
*   *opposite* direction. I.e. if inBaseQty_ is True, returns the quote token flow
*   for the swap. And vice versa..
*
* @dev The fixed-point result approximates the real valued formula with close but
*   directionally unpredicable precision. It could be slightly above or slightly
*   below. In the case of zero flows this could be substantially over. This
*   function should not be used in any context with strict directional boundness
*   requirements. */
// uint128 denomFlow = calcLimitFlows(curve, swapQty, inBaseQty, limitPrice);
// return invertFlow(activeLiquidity(curve), curve.priceRoot_,
//
//	   denomFlow, isBuy, inBaseQty);
//	}
func (c curveState) calcLimitCounter(swapQty *big.Int, inBaseQty bool, limitPrice *big.Int) *big.Int {
	// bool isBuy = limitPrice > curve.priceRoot_;
	// var isBuy = false
	// if limitPrice.Cmp(c.priceRoot) == 1 {
	// 	isBuy = true
	// }

	return nil
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
// function calcLimitFlows (CurveState memory curve, uint128 swapQty,
// bool inBaseQty, uint128 limitPrice)
// internal pure returns (uint128) {
// uint128 limitFlow = calcLimitFlows(curve, inBaseQty, limitPrice);
// return limitFlow > swapQty ? swapQty : limitFlow;
// }

// function calcLimitFlows (CurveState memory curve, bool inBaseQty,
// uint128 limitPrice) private pure returns (uint128) {
// uint128 liq = activeLiquidity(curve);
// return inBaseQty ?
// deltaBase(liq, curve.priceRoot_, limitPrice) :
// deltaQuote(liq, curve.priceRoot_, limitPrice);
// }
func (c curveState) calcLimitFlows(inBaseQty bool, limitPrice *big.Int) (*big.Int, error) {
	liq, err := c.activeLiquidity()
	if err != nil {
		return nil, err
	}
	if inBaseQty {

	}

	return nil, nil
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
	if inBaseQty {
		return mulQ64(liq, price)
	}

	return divQ64(liq, price)
}

// function reserveAtPrice (uint128 liq, uint128 price, bool inBaseQty)
// internal pure returns (uint128) {
// return (inBaseQty ?
// 			uint256(FixedPoint.mulQ64(liq, price)) :
// 			uint256(FixedPoint.divQ64(liq, price))).toUint128();
// }

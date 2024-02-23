package ambient

import (
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
}

func (s *PoolSimulator) CalcAmountOut(
	param poolpkg.CalcAmountOutParams,
) (*poolpkg.CalcAmountOutResult, error) {
	return nil, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {

}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

// /* @notice Swaps between two tokens within a single liquidity pool.
// *
// * @dev This is the most gas optimized swap call, since it avoids calling out to any
// *      proxy contract. However there's a possibility in the future that this call
// *      path could be disabled to support upgraded logic. In which case the caller
// *      should be able to swap through using a userCmd() call on the HOT_PATH proxy
// *      call path.
// *
// * @param base The base-side token of the pair. (For native Ethereum use 0x0)
// * @param quote The quote-side token of the pair.
// * @param poolIdx The index of the pool type to execute on.
// * @param isBuy If true the direction of the swap is for the user to send base tokens
// *              and receive back quote tokens.
// * @param inBaseQty If true the quantity is denominated in base-side tokens. If not
// *                  use quote-side tokens.
// * @param qty The quantity of tokens to swap. End result could be less if the pool
// *            price reaches limitPrice before exhausting.
// * @param tip A user-designated liquidity fee paid to the LPs in the pool. If set to
// *            0, just defaults to the standard pool rate. Otherwise represents the
// *            proposed LP fee in units of 1/1,000,000. Not used in standard swap
// *            calls, but may be used in certain permissioned or dynamic fee pools.
// * @param limitPrice The worse price the user is willing to pay on the margin. Swap
// *                   will execute up to this price, but not any worse. Average fill
// *                   price will always be equal or better, because this is calculated
// *                   at the marginal unit of quantity.
// * @param minOut The minimum output the user expects from the swap. If less is
// *               returned, the transaction will revert. (Alternatively if the swap
// *               is fixed in terms of output, this is the maximum input.)
// * @param reserveFlags Bitwise flags to indicate if the user wants to pay/receive in
// *                     terms of surplus collateral balance held at the dex contract.
// *                          0x1 - Base token is paid/received from surplus collateral
// *                          0x2 - Quote token is paid/received from surplus collateral
// * @return The token base and quote token flows associated with this swap action.
// *         (Negative indicates a credit paid to the user, positive a debit collected
// *         from the user) */
// function swap (address base, address quote,
// uint256 poolIdx, bool isBuy, bool inBaseQty, uint128 qty, uint16 tip,
// uint128 limitPrice, uint128 minOut,
// uint8 reserveFlags) reEntrantLock public payable
// returns (int128 baseQuote, int128 quoteFlow) {
// // By default the embedded hot-path is enabled, but protocol governance can
// // disable by toggling the force proxy flag. If so, users should point to
// // swapProxy.
// require(hotPathOpen_);
// return swapExecute(base, quote, poolIdx, isBuy, inBaseQty, qty, tip,
// 		limitPrice, minOut, reserveFlags);
// }

func (s *PoolSimulator) swap(
	base string, quote string, poolIdx *big.Int, isBuy bool, inBaseQty bool, qty *big.Int, tip uint16,
	limitPrice *big.Int, minOut *big.Int, reserveFlags uint8,
) (baseQuote, quoteFlow *big.Int) {
	// require(hotPathOpen_); (Skipping)

	return s.swapExecute(base, quote, poolIdx, isBuy, inBaseQty, qty, tip,
		limitPrice, minOut, reserveFlags)
}

func (s *PoolSimulator) swapExecute(
	base string, quote string, poolIdx *big.Int, isBuy bool, inBaseQty bool, qty *big.Int, tip uint16,
	limitPrice *big.Int, minOut *big.Int, reserveFlags uint8,
) (baseQuote, quoteFlow *big.Int) {
	// preparePoolCntx: query the pool, add poolTip, verify permit swap
	// PoolSpecs.PoolCursor memory pool = preparePoolCntx(base, quote, poolIdx, poolTip, isBuy, inBaseQty, qty);

	// Chaining.PairFlow memory flow = swapDir(pool, isBuy, inBaseQty, qty, limitPrice);
	// (baseFlow, quoteFlow) = (flow.baseFlow_, flow.quoteFlow_);

	// pivotOutFlow(flow, minOutput, isBuy, inBaseQty);
	// settleFlows(base, quote, flow.baseFlow_, flow.quoteFlow_, reserveFlags);
	// accumProtocolFees(flow, base, quote);

	return nil, nil
}

/* @notice Wrapper call to setup the swap directive object and call the swap logic in
*         the MarketSequencer mixin. */
func (s *PoolSimulator) swapDir(p swapPool, isBuy bool, inBaseQty bool, qty *big.Int, limitPrice *big.Int) (pairFlow, error) {
	// Directives.SwapDirective memory dir;
	// dir.isBuy_ = isBuy;
	// dir.inBaseQty_ = inBaseQty;
	// dir.qty_ = qty;
	// dir.limitPrice_ = limitPrice;
	// dir.rollType_ = 0;
	dir := swapDirective{
		isBuy:      isBuy,
		inBaseQty:  inBaseQty,
		qty:        qty,
		limitPrice: limitPrice,
		rollType:   0,
	}

	// return swapOverPool(dir, pool);
	return s.swapOverPool(dir, p)
}

/* @notice Performs a single swap over the pool.
* @param dir The user-specified directive governing the size, direction and limit
*            price of the swap to be performed.
* @param pool The pre-loaded speciication and hash of the pool to be swapped against.
* @return flow The net token flows generated by the swap. */
func (s *PoolSimulator) swapOverPool(dir swapDirective, p swapPool) (pairFlow, error) {
	// snapCurve: get curve from the pool hash
	// CurveMath.CurveState memory curve = snapCurve(pool.hash_);
	// sweepSwapLiq(flow, curve, curve.priceRoot_.getTickAtSqrtRatio(), dir, pool);
	// commitCurve(pool.hash_, curve);

	return pairFlow{}, nil
}

/* @notice Executes the pending swap through the order book, adjusting the
*         liquidity curve and level book as needed based on the swap's impact.
*
* @dev This is probably the most complex single function in the codebase. For
*      small local moves, which don't cross extant levels in the book, it acts
*      like a constant-product AMM curve. For large swaps which cross levels,
*      it iteratively re-adjusts the AMM curve on every level cross, and performs
*      the necessary book-keeping on each crossed level entry.
*
* @param accum The accumulator for the flows generated by the executable swap.
*              The realized flows on the swap will be written into the memory-based
*              accumulator fields of this struct. The caller is responsible for
*              ultaimtely paying and collecting those flows.
* @param curve The starting liquidity curve state. Any changes created by the
*              swap on this struct are updated in memory. But the caller is
*              responsible for committing the final state to EVM storage.
* @param midTick The price tick associated with the current price on the curve.
* @param swap The user specified directive governing the size, direction and limit
*             price of the swap to be executed.
* @param pool The pool's market specification notably its swap fee rate and the
*             protocol take rate. */
func (s *PoolSimulator) sweepSwapLiq(
	accum pairFlow,
	curve curveState,
	midTick int24,
	swap swapDirective,
	p swapPool,
) {
	// TODO:
	// require(swap.isBuy_ ? curve.priceRoot_ <= swap.limitPrice_ :
	// 	curve.priceRoot_ >= swap.limitPrice_, "SD");

	doMore := true

	for doMore {
		bumpTick, spillsOver := s.pinBitmap(p.hash, swap.isBuy, midTick)
		curve.swapToLimit
	}
}

/* @notice Finds an inner-bound conservative liquidity tick boundary based on
*   the terminus map at a starting tick point. Because liquidity actually bumps
*   at the bottom of the tick, the result is assymetric on direction. When seeking
*   an upper barrier, it'll be the tick that we cross into. For lower barriers, it's
*   the tick that we cross out of, and therefore could even be the starting tick.
*
* @dev For gas efficiency this method only looks at a previously loaded terminus
*   bitmap. Often for moves of that size we don't even need to look past the
*   terminus boundary. So there's no point doing a mezzanine layer seek unless we
*   end up needing it.
*
* @param poolIdx The hash key associated with the pool being queried.
* @param isUpper - If true indicates that we're looking for an upper boundary.
* @param startTick - The current tick index that we're finding the boundary from.
*
* @return boundTick - The tick index that we can conservatively move to without
*    potentially hitting any currently active liquidity bump points.
* @return isSpill - If true indicates that the boundary represents the end of the
*    inner terminus bitmap neighborhood. Based on this we have to actually check whether
*     we've reached teh true end of the liquidity range, or just the end of the known
*     neighborhood.  */
func (s *PoolSimulator) pinBitmap(poolIdx string, isUpper bool, startTick int24) (boundTick int24, isSpill bool) {
	// uint256 termBitmap = terminusBitmap(poolIdx, startTick);

	// uint16 shiftTerm = startTick.termBump(isUpper);
	// int16 tickMezz = startTick.mezzKey();
	// (boundTick, isSpill) = pinTermMezz
	// (isUpper, shiftTerm, tickMezz, termBitmap);

	return 0, false
}

package ambient

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient/types"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	curve *curveState
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
func (s *PoolSimulator) swapDir(p *swapPool, isBuy bool, inBaseQty bool, qty *big.Int, limitPrice *big.Int) (pairFlow, error) {
	// Directives.SwapDirective memory dir;
	// dir.isBuy_ = isBuy;
	// dir.inBaseQty_ = inBaseQty;
	// dir.qty_ = qty;
	// dir.limitPrice_ = limitPrice;
	// dir.rollType_ = 0;
	dir := &swapDirective{
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
func (s *PoolSimulator) swapOverPool(dir *swapDirective, p *swapPool) (pairFlow, error) {
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
	accum *pairFlow,
	curve *curveState,
	midTick types.Int24,
	swap *swapDirective,
	p swapPool,
) error {
	// TODO:
	// require(swap.isBuy_ ? curve.priceRoot_ <= swap.limitPrice_ :
	// 	curve.priceRoot_ >= swap.limitPrice_, "SD");

	// Keep iteratively executing more quantity until we either reach our limit price
	// or have zero quantity left to execute.
	doMore := true

	for doMore {
		// Swap to furthest point we can based on the local bitmap. Don't bother
		// seeking a bump outside the local neighborhood yet, because we're not sure
		// if the swap will exhaust the bitmap.
		bumpTick, spillsOver, err := s.pinBitmap(p.hash, swap.isBuy, midTick)
		if err != nil {
			return err
		}
		curve.swapToLimit(accum, swap, p, bumpTick)

		// The swap can be in one of four states at this point: 1) qty exhausted,
		// 2) limit price reached, 3) bump or barrier point reached on the curve.
		// The former two indicate the swap is complete. The latter means we have to
		// find the next bump point and possibly adjust AMM liquidity.
		doMore = hasSwapLeft(curve, swap)
		if doMore {
			// The spillsOver variable indicates that we reached stopped because we
			// reached the end of the local bitmap, rather than actually hitting a
			// level bump. Therefore we should query the global bitmap, find the next
			// bump point, and keep swapping across the constant-product curve until
			// if/when we hit that point.
			if spillsOver {
				// int24 liqTick = seekMezzSpill(pool.hash_, bumpTick, swap.isBuy_);
				liqTick, err := seekMezzSpill(p.hash, bumpTick, swap.isBuy)
				if err != nil {
					return err
				}

				// bool tightSpill = (bumpTick == liqTick);
				tightSpill := bumpTick == liqTick
				//     bumpTick = liqTick;
				bumpTick = liqTick

				// In some corner cases the local bitmap border also happens to
				// be the next bump point. If so, we're done with this inner section.
				// Otherwise, we keep swapping since we still have some distance on
				// the curve to cover until we reach a bump point.
				// if (!tightSpill) {
				if !tightSpill {
					// TODO: what the h** is pool.head_
					// 	curve.swapToLimit(accum, swap, pool.head_, bumpTick);
					curve.swapToLimit(accum, swap, p, bumpTick)
					// 	doMore = hasSwapLeft(curve, swap);
					doMore = hasSwapLeft(curve, swap)
				}
			}

			// Perform book-keeping related to crossing the level bump, update
			// the locally tracked tick of the curve price (rather than wastefully
			// we calculating it since we already know it), then begin the swap
			// loop again.
			if doMore {
				mTick, err := knockInTick(accum, bumpTick, curve, swap, p.hash)
				if err != nil {
					return nil
				}
				midTick = mTick
			}
		}
	}

	return nil
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
func (s *PoolSimulator) pinBitmap(poolIdx string, isUpper bool, startTick types.Int24) (types.Int24, bool, error) {
	// uint256 termBitmap = terminusBitmap(poolIdx, startTick);
	termBitmap := terminusBitmap(poolIdx, startTick)

	// uint16 shiftTerm = startTick.termBump(isUpper);
	shiftTerm := termBump(startTick, isUpper)

	// int16 tickMezz = startTick.mezzKey();
	tickMezz := startTick.MezzKey()

	// (boundTick, isSpill) = pinTermMezz
	// (isUpper, shiftTerm, tickMezz, termBitmap);
	boundTick, isSpill, err := pinTermMezz(isUpper, shiftTerm, tickMezz, termBitmap)
	if err != nil {
		return 0, false, err
	}

	return boundTick, isSpill, nil
}

/* @notice Performs all the necessary book keeping related to crossing an extant
*         level bump on the curve.
*
* @dev Note that this function updates the level book data structure directly on
*      the EVM storage. But it only updates the liquidity curve state *in memory*.
*      This is for gas efficiency reasons, as the same curve struct may be updated
*      many times in a single swap. The caller must take responsibility for
*      committing the final curve state back to EVM storage.
*
* @params bumpTick The tick index where the bump occurs.
* @params isBuy The direction the bump happens from. If true, curve's price is
*               moving through the bump starting from a lower price and going to a
*               higher price. If false, the opposite.
* @params curve The pre-bump state of the local constant-product AMM curve. Updated
*               to reflect the liquidity added/removed from rolling through the
*               bump.
* @param swap The user directive governing the size, direction and limit price of the
*             swap to be executed.
* @param poolHash The key hash mapping to the pool we're executive over.
*
* @return The tick index that the curve and its price are living in after the call
*         completes. */
func knockInTick(accum *pairFlow, bumpTick types.Int24, curve *curveState, swap *swapDirective, poolHash string) (types.Int24, error) {
	// if (!Bitmaps.isTickFinite(bumpTick)) { return bumpTick; }
	if !isTickFinite(bumpTick) {
		return bumpTick, nil
	}

	// bumpLiquidity(curve, bumpTick, swap.isBuy_, poolHash);
	bumpLiquidity(curve, bumpTick, swap.isBuy, poolHash)

	// (int128 paidBase, int128 paidQuote, uint128 burnSwap) =
	// curve.shaveAtBump(swap.inBaseQty_, swap.isBuy_, swap.qty_);
	paidBase, paidQuote, burnSwap, err := shaveAtBump(curve, swap.inBaseQty, swap.isBuy, swap.qty)
	if err != nil {
		return 0, err
	}

	// accum.accumFlow(paidBase, paidQuote);
	accum.accumFlow(paidBase, paidQuote)

	// // burn down qty from shaveAtBump is always validated to be less than remaining swap.qty_
	// // so this will never underflow
	// swap.qty_ -= burnSwap;
	swap.qty.Sub(swap.qty, burnSwap)

	// // When selling down, the next tick leg actually occurs *below* the bump tick
	// // because the bump barrier is the first price on a tick.
	// return swap.isBuy_ ?
	// bumpTick :
	// bumpTick - 1; // Valid ticks are well above {min(int128)-1}, so will never underflow
	// }
	if swap.isBuy {
		return bumpTick, nil
	}

	return bumpTick - 1, nil
}

/* @notice Performs the book-keeping related to crossing a concentrated liquidity
*         bump tick, and adjusts the in-memory curve object with the change of
*         AMM liquidity. */
func bumpLiquidity(curve *curveState, bumpTick types.Int24, isBuy bool, poolHash string) {
	// (int128 liqDelta, bool knockoutFlag) =
	// crossLevel(poolHash, bumpTick, isBuy, curve.concGrowth_);
	liqDelta, knockoutFlag := crossLevel(poolHash, bumpTick, isBuy, curve.concGrowth)

	// curve.concLiq_ = curve.concLiq_.addDelta(liqDelta);
	curve.concLiq = addDelta(curve.concLiq, liqDelta)

	// if (knockoutFlag) {
	if knockoutFlag {
		// 		int128 knockoutDelta = callCrossFlag
		// 		(poolHash, bumpTick, isBuy, curve.concGrowth_);
		knockoutDelta := callCrossFlag(poolHash, bumpTick, isBuy, curve.concGrowth)

		// 		curve.concLiq_ = curve.concLiq_.addDelta(knockoutDelta);
		curve.concLiq = addDelta(curve.concLiq, knockoutDelta)
	}
	// }
}

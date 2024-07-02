package ambient

import (
	"math/big"
)

/* @notice Called when a given knockout pivot is crossed. Performs the book-keeping
*         related to reseting the pivot object and committing the Merkle history.
*         Does *not* adjust the liquidity on the bump point or curve, caller is
*         responsible for that upstream.
*
* @dev This function must only be called *after* the AMM curve has crossed the
*      tick and fee odometer on the tick has been updated to reflect the update.
*
* @param pool The hash index of the AMM pool.
* @param isBid If true, indicates that it's a bid pivot being knocked out (i.e.
*              that price is moving down through the pivot)
* @param tick The tick index of the knockout pivot.
* @param feeMileage The in range fee mileage at the point the pivot was crossed. */
func crossKnockOut(pool string, isBid bool, tick Int24, feeGlobal uint64) {
	// bytes32 lvlKey = KnockoutLiq.encodePivotKey(pool, isBid, tick);
	lvlKey := encodePivotKey(pool, isBid, tick)

	// KnockoutLiq.KnockoutPivot storage pivot = knockoutPivots_[lvlKey];
	pivot := knockoutPivots(lvlKey)

	// KnockoutLiq.KnockoutMerkle storage merkle = knockoutMerkles_[lvlKey];
	_ = knockoutMerkles(lvlKey)

	// unmarkPivot(pool, isBid, tick);
	unmarkPivot(pool, isBid, tick)

	// uint64 feeRange = knockoutRangeLiq(pool, pivot, isBid, tick, feeGlobal);
	_ = knockoutRangeLiq(pool, &pivot, isBid, tick, feeGlobal)

	// WONTDO
	// merkle.commitKnockout(pivot, feeRange);
	// emit CrocKnockoutCross(pool, tick, isBid, merkle.pivotTime_, merkle.feeMileage_,
	// 		   KnockoutLiq.commitEntropySalt());
	// pivot.deletePivot(); // Nice little SSTORE refund for the swapper

}

/* @notice Removes the mark on the book level related to the presence of knockout
*         liquidity. */
func unmarkPivot(pool string, isBid bool, tick Int24) {
	//     BookLevel storage lvl = fetchLevel(pool, tick);
	lvl := fetchLevel(pool, tick)

	//     if (isBid) {
	//         lvl.bidLots_ = lvl.bidLots_ & ~uint96(0x1);
	//     } else {
	//         lvl.askLots_ = lvl.askLots_ & ~uint96(0x1);
	//     }
	if isBid {
		lvl.bidLots.And(lvl.bidLots, new(big.Int).Not(big0x1))
	} else {
		lvl.askLots.And(lvl.askLots, new(big.Int).Not(big0x1))
	}
}

/* @notice Removes the liquidity at the AMM curve's bump points as part of a pivot
 *         being knocked out by a level cross. */
func knockoutRangeLiq(pool string, pivot *knockoutPivot, isBid bool, tick Int24, feeGlobal uint64) uint64 {
	// int24 offset = int24(uint24(pivot.rangeTicks_));
	offset := Int24(Uint24(pivot.rangeTicks))

	// int24 priceTick = isBid ? tick-1 : tick;
	var priceTick Int24
	if isBid {
		priceTick = tick - 1
	} else {
		priceTick = tick
	}

	// int24 lowerTick = isBid ? tick : tick - offset;
	var lowerTick Int24
	if isBid {
		lowerTick = tick
	} else {
		lowerTick = tick - offset
	}

	// int24 upperTick = !isBid ? tick : tick + offset;
	var upperTick Int24
	if !isBid {
		upperTick = tick
	} else {
		upperTick = tick - offset
	}

	// feeRange = removeBookLiq(pool, priceTick, lowerTick, upperTick,
	// 		  pivot.lots_, feeGlobal);
	feeRange := removeBookLiq(pool, priceTick, lowerTick, upperTick, pivot.lots, feeGlobal)

	return feeRange
}

/* @notice Call when removing liquidity associated with a specific range order.
*         Decrements the associated tick levels as necessary.
*
* @param poolIdx - The index of the pool the liquidity is being removed from.
* @param midTick - The tick index associated with the current price of the AMM curve
* @param bidTick - The tick index for the lower bound of the range order.
* @param askTick - The tick index for the upper bound of the range order.
* @param liq - The amount of liquidity being added by the range order.
* @param feeGlobal - The up-to-date global fee rewards growth accumulator.
*    Represented as 128-bit fixed point growth rate.
*
* @return feeOdometer - Returns the current fee reward accumulator value for the
*    range specified by the order. Note that this returns the accumulated rewards
*    from the range history, including *before* the order was added. It's the
*    downstream user's responsibility to adjust this value with the odometer clock
*    from addBookLiq to correctly calculate the rewards accumulated over the
*    lifetime of the order. */
func removeBookLiq(poolIdx string, midTick Int24, bidTick Int24, askTick Int24, lots *big.Int, feeGlobal uint64) uint64 {
	// bool deleteBid = removeBid(pooldx, bidTick, lots);
	deleteBid := removeBid(poolIdx, bidTick, lots)

	// bool deleteAsk = removeAsk(poolIdx, askTick, lots);
	deleteAsk := removeAsk(poolIdx, askTick, lots)

	// feeOdometer = clockFeeOdometer(poolIdx, midTick, bidTick, askTick, feeGlobal);
	feeOdometer := clockFeeOdometer(poolIdx, midTick, bidTick, askTick, feeGlobal)

	if deleteBid {
		deleteLevel(poolIdx, bidTick)
	}

	if deleteAsk {
		deleteLevel(poolIdx, askTick)
	}

	return feeOdometer
}

// if (deleteBid) { deleteLevel(poolIdx, bidTick); }
// if (deleteAsk) { deleteLevel(poolIdx, askTick); }
// }

/* @notice Decrements bid liquidity on a level, and also removes the level from
*          the tick bitmap if necessary. */
func removeBid(poolIdx string, tick Int24, subLots *big.Int) bool {
	// BookLevel storage lvl = fetchLevel(poolIdx, tick);
	lvl := fetchLevel(poolIdx, tick)

	// uint96 prevLiq = lvl.bidLots_;
	prevLiq := lvl.bidLots

	// uint96 newLiq = prevLiq.minusLots(subLots);
	newLiq := minusLots(prevLiq, subLots)

	// // A level should only be marked inactive in the tick bitmap if *both* bid and
	// // ask liquidity are zero.
	// lvl.bidLots_ = newLiq;
	lvl.bidLots = newLiq

	// if (newLiq == 0 && lvl.askLots_ == 0) {
	// 		forgetTick(poolIdx, tick);
	// 		return true;
	// }
	// return false;
	// }
	if newLiq.Cmp(big0) == 0 && lvl.askLots.Cmp(big0) == 0 {
		forgetTick(poolIdx, tick)
		return true
	}

	return false
}

/* @notice Decrements ask liquidity on a level, and also removes the level from
*          the tick bitmap if necessary. */
func removeAsk(poolIdx string, tick Int24, subLots *big.Int) bool {
	lvl := fetchLevel(poolIdx, tick)
	prevLiq := lvl.askLots
	newLiq := minusLots(prevLiq, subLots)
	lvl.askLots = newLiq

	if newLiq.Cmp(big0) == 0 && lvl.bidLots.Cmp(big0) == 0 {
		forgetTick(poolIdx, tick)
		return true
	}

	return false
}

// /* @notice Deletes the level at the tick. */
func deleteLevel(_ string, _ Int24) {
	// TODO: impl this
	// delete levels_[keccak256(abi.encodePacked(poolIdx, tick))];
}

// function deleteLevel (bytes32 poolIdx, int24 tick) private {
// }

func knockoutPivots(_ string) knockoutPivot {
	// TODO: get mapping
	return knockoutPivot{}
}

func knockoutMerkles(_ string) knockoutMerkle {
	// TODO: get mapping
	return knockoutMerkle{}
}

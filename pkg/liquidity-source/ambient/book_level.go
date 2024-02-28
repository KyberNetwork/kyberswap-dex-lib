package ambient

import (
	"encoding/binary"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

/* Book level structure exists one-to-one on a tick basis (though could possibly be
* zero-valued). For each tick we have to track three values:
*    bidLots_ - The change to concentrated liquidity that's added to the AMM curve when
*               price moves into the tick from below, and removed when price moves
*               into the tick from above. Denominated in lot-units which are 1024 multiples
*               of liquidity units.
*    askLots_ - The change to concentrated liquidity that's added to the AMM curve when
*               price moves into the tick from above, and removed when price moves
*               into the tick from below. Denominated in lot-units which are 1024 multiples
*               of liquidity units.
*    feeOdometer_ - The liquidity fee rewards accumulator that's checkpointed
*       whenever the price crosses the tick boundary. Used to calculate the
*       cumulative fee rewards on any arbitrary lower-upper tick range. This is
*       generically represented as a per-liquidity unit 128-bit fixed point
*       cumulative growth rate. */

/* @notice Called when the curve price moves through the tick boundary. Performs
*         the necessary accumulator checkpointing and deriving the liquidity bump.
*
* @dev    Note that this function call is *not* idempotent. It's the callers
*         responsibility to only call once per tick cross direction. Otherwise
*         behavior is undefined. This is safe to call with non-initialized zero
*         ticks but should generally be avoided for gas efficiency reasons.
*
* @param poolIdx - The hash index of the pool being traded on.
* @param tick - The 24-bit tick index being crossed.
* @param isBuy - If true indicates that price is crossing the tick boundary from
*                 below. If false, means tick is being crossed from above.
* @param feeGlobal - The up-to-date global fee reward accumulator value. Used to
*                    checkpoint the tick rewards for calculating accumulated rewards
*                    in a range. Represented as 128-bit fixed point cumulative
*                    growth rate per unit of liquidity.
*
* @return liqDelta - The net change in concentrated liquidity that should be applied
*                    to the AMM curve following this level cross.
* @return knockoutFlag - Indicates that the liquidity of the cross level has a
*                        knockout flag toggled. Upstream caller should handle
*                        appropriately */
func crossLevel(poolIdx string, tick types.Int24, isBuy bool, feeGlobal uint64) (*big.Int, bool) {
	// 	 BookLevel storage lvl = fetchLevel(poolIdx, tick);
	lvl := fetchLevel(poolIdx, tick)

	// 	 int128 crossDelta = LiquidityMath.netLotsOnLiquidity
	// 		 (lvl.bidLots_, lvl.askLots_);
	crossDelta := netLotsOnLiquidity(lvl.bidLots, lvl.askLots)

	// 	 liqDelta = isBuy ? crossDelta : -crossDelta;
	liqDelta := new(big.Int)
	if isBuy {
		liqDelta.Set(crossDelta)
	} else {
		liqDelta.Set(crossDelta)
		liqDelta.Neg(liqDelta)
	}

	// 	 if (feeGlobal != lvl.feeOdometer_) {
	// 		 lvl.feeOdometer_ = feeGlobal - lvl.feeOdometer_;
	// 	 }
	if feeGlobal != lvl.feeOdometer {
		lvl.feeOdometer = feeGlobal - lvl.feeOdometer
	}

	// 	 knockoutFlag = isBuy ?
	// 		 lvl.askLots_.hasKnockoutLiq() :
	// 		 lvl.bidLots_.hasKnockoutLiq();
	//  }
	var knockOutFlag bool
	if isBuy {
		knockOutFlag = hasKnockoutLiq(lvl.askLots)
	} else {
		knockOutFlag = hasKnockoutLiq(lvl.bidLots)
	}

	return liqDelta, knockOutFlag
}

// 	 function crossLevel (bytes32 poolIdx, int24 tick, bool isBuy, uint64 feeGlobal)
// 	 internal returns (int128 liqDelta, bool knockoutFlag) {

/* @notice Checks if an aggergate lots counter contains a knockout liquidity component
 *         by checking the least significant bit.
 *
 * @dev    Note that it's critical that the sum *total* of knockout lots on any
 *         given level be an odd number. Don't add two odd knockout lots together
 *         without renormalzing, because they'll sum to an even lot quantity. */
func hasKnockoutLiq(lots *big.Int) bool {
	return new(big.Int).And(lots, KNOCKOUT_FLAG_MASK).Cmp(big0) == 1
}

//  function hasKnockoutLiq (uint96 lots) internal pure returns (bool) {
//     return lots & KNOCKOUT_FLAG_MASK > 0;
// }

/* @notice Retrieves a storage pointer to the level associated with the tick. */
func fetchLevel(poolIdx string, tick types.Int24) BookLevel {
	//     return levels_[keccak256(abi.encodePacked(poolIdx, tick))];
	tmp := make([]byte, 4)
	binary.LittleEndian.PutUint32(tmp, uint32(tick))
	s := abi.EncodePacked([]byte(poolIdx), tmp)

	return levels(string(s))
}

// TODO: how to get levels
func levels(_ string) BookLevel {
	return BookLevel{}
}

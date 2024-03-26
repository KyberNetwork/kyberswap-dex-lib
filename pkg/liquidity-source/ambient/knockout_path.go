package ambient

import (
	"math/big"
)

/* @notice Called when a knockout pivot is crossed.
*
* @dev Since this contract is a proxy sidecar, this method needs to be marked
*      payable even though it doesn't directly handle msg.value. Otherwise it will
*      fail on any. Because of this, this contract should never be used in any other
*      context besides a proxy sidecar to CrocSwapDex.
*
* @param pool The hash index of the pool.
* @param tick The 24-bit index of the tick where the knockout pivot exists.
* @param isBuy If true indicates that the swap direction is a buy.
* @param feeGlobal The global fee odometer for 1 hypothetical unit of liquidity fully
*                  in range since the inception of the pool.
*
* @return Returns the net additional amount the curve liquidity should be adjusted by.
*         Currently this always returns zero, because a liquidity knockout will never change
*         active liquidity on a curve. But by leaving this function return type it leaves open
*         the possibility in future upgrades of alternative types of dynamic liquidity that
*         do change active curve liquidity when crossed */
func crossCurveFlag(pool string, tick Int24, isBuy bool, feeGlobal uint64) *big.Int {
	// If swap is a sell, then implies we're crossing a resting bid and vice versa
	// 	 bool bidCross = !isBuy;
	bidCross := !isBuy

	// 	 crossKnockout(pool, bidCross, tick, feeGlobal);
	crossKnockOut(pool, bidCross, tick, feeGlobal)
	// 	 return 0;
	return big.NewInt(0)
}

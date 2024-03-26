package ambient

import (
	"fmt"
	"math/big"
)

/* @notice Defines a currently active knockout liquidity bump point that exists on
 *         a specific AMM curve at a specific tick/direction.
 *
 * @param lots_ The total number of lots active in the knockout pivot. Note that this
 *              number should always be included in the corresponding LevelBook lots.
 *
 * @param pivotTime_ The block time the first liquidity was created on the pivot
 *                   point. This resets every time the knockout is crossed, and is
 *                   therefore used to distinguish tranches of liquidity that were
 *                   added at the same tick but with different knockout times.
 *
 * @param rangeTicks_ The number of ticks wide the range order for the knockout
 *                    tranche. Unlike traditional concentrated liquidity, all knockout
 *                    liquidity in the same tranche must have the same width. This is
 *                    used to determine what counter-side tick to decrement liquidity
 *                    on when knocking out an order. */
type knockoutPivot struct {
	//     uint96 lots_;
	lots *big.Int
	//     uint32 pivotTime_;
	pivotTime uint32
	//     uint16 rangeTicks_;
	rangeTicks uint16
}

/* @notice Stores a cryptographically provable history of previous knockout events
 *         at a given tick/direction.
 *
 * @dev To avoid unnecessary SSTORES, we Merkle at the same location instead of
 *      growing an array. This allows users trying to claim a previously knockout
 *      position to post a Merkle proof. (And since the underlying liquidity is
 *      computable even without this proof, the only loss for those that don't are the
 *      accumulated fees while the range liquidity was active.)
 *
 * @param merkleRoot_ The Merkle root of the prior entry in the chain.
 * @param pivotTime_ The pivot time of the last tranche to be knocked out at this tick
 * @param feeMileage_ The fee mileage for the range at the time the tranche was
 *                    knocked out. */
type knockoutMerkle struct {
	//     uint160 merkleRoot_;
	merkleRoot *big.Int
	//     uint32 pivotTime_;
	pivotTime uint32
	//     uint64 feeMileage_;
	feeMileage uint64
}

/* @notice Encodes a hash key for a given knockout pivot point.
* @param pool The hash index of the AMM pool.
* @param isBid If true indicates the knockout pivot is on the bid side.
* @param tick The tick index of the knockout pivot.
* @return Unique hash key mapping to the pivot struct. */
func encodePivotKey(pool string, isBid bool, tick Int24) string {
	// TODO: impl real one
	// 	 return keccak256(abi.encode(pool, isBid, tick));
	return fmt.Sprintf("%s-%t-%d", pool, isBid, tick)
}

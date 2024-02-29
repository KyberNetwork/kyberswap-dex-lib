package ambient

import (
	"encoding/binary"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"golang.org/x/crypto/sha3"
)

/* @notice Determines the next tick bump boundary tick starting using recursive
*   bitmap lookup. Follows the same up/down assymetry as pinBitmap(). Upper bump
*   is the tick being crossed *into*, lower bump is the tick being crossed *out of*
*
* @dev This is a much more gas heavy operation because it recursively looks
*   though all three layers of bitmaps. It should only be called if pinBitmap()
*   can't find the boundary in the terminus layer.
*
* @param poolIdx The hash key associated with the pool being queried.
* @param borderTick - The current tick that we want to seek a tick liquidity
*   boundary from. For defined behavior this tick must occur at the border of
*   terminus bitmap. For lower borders, must be the tick from the start of the byte.
*   For upper borders, must be the tick past the end of the byte. Any spill result
*   from pinTermMezz() is safe.
* @param isUpper - The direction of the boundary. If true seek an upper boundary.
*
* @return (int24) - The tick index of the next tick boundary with an active
*   liquidity bump. The result is assymetric boundary for upper/lower ticks. */
func seekMezzSpill(poolIdx string, borderTick Int24, isUpper bool) (Int24, error) {
	// 	 if (isUpper && borderTick == type(int24).max) { return type(int24).max; }
	if isUpper && borderTick == Int24Max {
		return Int24Max, nil
	}

	// 	 if (!isUpper && borderTick == type(int24).min) { return type(int24).min; }
	if !isUpper && borderTick == Int24Min {
		return Int24Min, nil
	}

	// 	 (uint8 lobbyBorder, uint8 mezzBorder) = rootsForBorder(borderTick, isUpper);
	lobbyBorder, mezzBorder := rootsForBorder(borderTick, isUpper)

	// 	 // Most common case is that the next neighboring bitmap on the border has
	// 	 // an active tick. So first check here to save gas in the hotpath.
	// 	 (int24 pin, bool spills) =
	// 		 seekAtTerm(poolIdx, lobbyBorder, mezzBorder, isUpper);
	pin, spills, err := seekAtTerm(poolIdx, lobbyBorder, mezzBorder, isUpper)
	if err != nil {
		return 0, nil
	}

	// 	 if (!spills) { return pin; }
	if !spills {
		return pin, nil
	}

	// Next check to see if we can find a neighbor in the mezzanine. This almost
	// always happens except for very sparse pools.
	// (pin, spills) =
	//     seekAtMezz(poolIdx, lobbyBorder, mezzBorder, isUpper);
	// if (!spills) { return pin; }
	pin, spills, err = seekAtMezz(poolIdx, lobbyBorder, mezzBorder, isUpper)
	if err != nil {
		return 0, err
	}
	if !spills {
		return pin, nil
	}

	// Finally iterate through the lobby layer.
	// return seekOverLobby(poolIdx, lobbyBorder, isUpper);
	return seekOverLobby(poolIdx, lobbyBorder, isUpper)
}

/* @notice Seeks the next tick bitmap by searching in the adjacent neighborhood. */
func seekAtTerm(poolIdx string, lobbyBit uint8, mezzBit uint8, isUpper bool) (Int24, bool, error) {
	//     uint256 neighborBitmap = terminus_
	//         [encodeTermWord(poolIdx, lobbyBit, mezzBit)];
	neighborBitmap := terminus(encodeTermWord(poolIdx, lobbyBit, mezzBit))

	// (uint8 termBit, bool spills) = neighborBitmap.bitAfterTrunc(0, isUpper);
	termBit, spills, err := bitAfterTrunc(neighborBitmap, 0, isUpper)
	if err != nil {
		return 0, false, err
	}

	//     if (spills) { return (0, true); }
	if spills {
		return 0, true, nil
	}

	//     return (Bitmaps.weldLobbyPosMezzTerm(lobbyBit, mezzBit, termBit), false);
	i := weldLobbyMezzTerm(int8(lobbyBit), mezzBit, termBit)
	return i, false, nil
}

/* @notice Seeks the next tick bitmap by searching in the current mezzanine
 *         neighborhood.
 * @dev This covers a span of 65 thousand ticks, so should capture most cases. */
func seekAtMezz(poolIdx string, lobbyBit uint8, mezzBorder uint8, isUpper bool) (Int24, bool, error) {
	// uint256 neighborMezz = mezzanine_
	// [encodeMezzWord(poolIdx, lobbyBit)];
	neighborMezz := mezzanine(encodeMezzWord(poolIdx, int8(lobbyBit)))

	// uint8 mezzShift = Bitmaps.bitRelate(mezzBorder, isUpper);
	mezzShift := bitRelate(mezzBorder, isUpper)

	// (uint8 mezzBit, bool spills) = neighborMezz.bitAfterTrunc(mezzShift, isUpper);
	mezzBit, spills, err := bitAfterTrunc(neighborMezz, uint16(mezzShift), isUpper)
	if err != nil {
		return 0, false, err
	}

	// if (spills) { return (0, true); }
	if spills {
		return 0, true, nil
	}

	// return seekAtTerm(poolIdx, lobbyBit, mezzBit, isUpper);
	return seekAtTerm(poolIdx, lobbyBit, mezzBit, isUpper)
}

/* @notice Encodes the hash key for the terminus neighborhood of the first 16-bits
 *         of a tick index. (This is all that's needed to determine terminus.) */
func encodeTermWord(poolIdx string, lobbyPos uint8, mezzPos uint8) string {
	// 	 int16 mezzIdx = Bitmaps.weldLobbyMezz
	// 		 (Bitmaps.uncastBitmapIndex(lobbyPos), mezzPos);
	mezzIdx := weldLobbyMezz(uncastBitmapIndex(lobbyPos), mezzPos)

	// 	 return keccak256(abi.encodePacked(poolIdx, mezzIdx));
	h := sha3.NewLegacyKeccak256()
	tmp := make([]byte, 2)
	binary.LittleEndian.PutUint16(tmp, uint16(mezzIdx))
	h.Write(abi.EncodePacked([]byte(poolIdx), tmp))

	return string(h.Sum(nil))
}

/* @notice Splits out the lobby bits and the mezzanine bits from the 24-bit price
*         tick index associated with the type of border tick used in seekMezzSpill()
*         call */
func rootsForBorder(borderTick Int24, isUpper bool) (uint8, uint8) {
	// Because pinTermMezz returns a border *on* the previous bitmap, we need to
	// decrement by one to get the seek starting point.
	// int24 pinTick = isUpper ? borderTick : (borderTick - 1);
	var pinTick Int24
	if isUpper {
		pinTick = borderTick
	} else {
		pinTick = borderTick - 1
	}

	// 	 lobbyBit = pinTick.lobbyBit();
	lobbyBit := pinTick.LobbyBit()

	// 	 mezzBit = pinTick.mezzBit();
	mezzBit := pinTick.MezzBit()

	return lobbyBit, mezzBit
}

/* @notice Encodes the hash key for the mezzanine neighborhood of the first 8-bits
 *         of a tick index. (This is all that's needed to determine mezzanine.) */
func encodeMezzWord(poolIdx string, lobbyPos int8) string {
	// return keccak256(abi.encodePacked(poolIdx, lobbyPos));
	h := sha3.NewLegacyKeccak256()
	h.Write(abi.EncodePacked([]byte(poolIdx), []byte{byte(lobbyPos)}))

	return string(h.Sum(nil))
}

/* @notice Used when the tick is not contained in the mezzanine. We walk through the
 *         the mezzanine tick bitmaps one by one until we find an active tick bit. */
func seekOverLobby(poolIdx string, lobbyBit uint8, isUpper bool) (Int24, error) {
	if isUpper {
		return seekLobbyUp(poolIdx, lobbyBit)
	}

	return seekLobbyDown(poolIdx, lobbyBit)
}

/* Unlike the terminus and mezzanine layer, we don't store a bitmap at the lobby
 * layer. Instead we iterate through the top-level bits until we find an active
 * mezzanine. This requires a maximum of 256 iterations, and can be gas intensive.
 * However moves at this level represent 65,000% price changes and are very rare. */
//  function seekLobbyUp (bytes32 poolIdx, uint8 lobbyBit)
func seekLobbyUp(poolIdx string, lobbyBit uint8) (Int24, error) {
	// 	 uint8 MAX_MEZZ = 0;
	var MAX_MEZZ uint8 = 0

	// 		 for (uint8 i = lobbyBit + 1; i > 0; ++i) {
	for i := lobbyBit + 1; i > 0; i++ {
		// (int24 tick, bool spills) = seekAtMezz(poolIdx, i, MAX_MEZZ, true);
		tick, spills, err := seekAtMezz(poolIdx, i, MAX_MEZZ, true)
		if err != nil {
			return 0, err
		}

		// if (!spills) { return tick; }
		if !spills {
			return tick, nil
		}
	}

	// 	 return Bitmaps.zeroTick(true);
	return zeroTick(true), nil
}

/* Same logic as seekLobbyUp(), but the inverse direction. */
func seekLobbyDown(poolIdx string, lobbyBit uint8) (Int24, error) {
	// 	 uint8 MIN_MEZZ = 255;
	var MIN_MEZZ = 255

	// 		 for (uint8 i = lobbyBit - 1; i < 255; --i) {
	for i := lobbyBit - 1; i < 255; i-- {
		// 			 (int24 tick, bool spills) = seekAtMezz(poolIdx, i, MIN_MEZZ, false);
		tick, spills, err := seekAtMezz(poolIdx, i, uint8(MIN_MEZZ), false)
		if err != nil {
			return 0, err
		}
		// 			 if (!spills) { return tick; }
		if !spills {
			return tick, nil
		}
	}

	// 	 return Bitmaps.zeroTick(false);
	return zeroTick(false), nil
}

/* Tick positions are stored in three layers of 8-bit/256-slot bitmaps. Recursively
* they indicate whether any given 24-bit tick index is active.

* The first layer (lobby) represents the 8-bit tick root. If we did store this
* layer, we'd only need a single 256-bit bitmap per pool. However we do *not*
* store this layer, because it adds an unnecessary SLOAD/SSTORE operation on
* almost all operations. Instead users can query this layer by checking whether
* mezzanine key is set for each bit. The tradeoff is that lobby bitmap queries
* are no longer O(1) random access but O(N) seeks. However at most there are 256
* SLOAD on a lobby-layer seek, and spills at the lobby layer are rare (moving
* between multiple lobby bits requires a 65,000% price change). This gas tradeoff
*  is virtually always justified.
*
* The second layer (mezzanine) maps whether each 16-bit tick root is set. An
* entry will be set if and only if *any* tick index in the 8-bit range is set.
* Because there are 256^2 slots, this is represented as a map from the first 8-
* bits in the root to individual 8-bit/256-slot bitmaps for the middle 8-bits
* at that root.
*
* The final layer (terminus) directly maps whether individual tick indices are
* set. Because there are 256^3 possible slots, this is represnted as a mapping
* from the first 16-bit tick root to individual 8-bit/256-slot bitmaps of the
* terminal 8-bits within that root. */

/* @notice Returns the associated bitmap for the terminus position (bottom layer)
*         of the tick index.
* @param poolIdx The hash key associated with the pool being queried.
* @param tick A price tick index within the neighborhood that we want the bitmap for.
* @return The bitmap of the 256-tick neighborhood. */
func terminusBitmap(poolIdx string, tick Int24) *big.Int {
	// bytes32 idx = encodeTerm(poolIdx, tick);
	idx := encodeTerm(poolIdx, tick)

	// return terminus_[idx];
	return terminus(idx)
}

/* @notice Encodes the hash key for the terminus neighborhood of the tick. */
func encodeTerm(poolIdx string, tick Int24) string {
	// 	int16 wordPos = tick.mezzKey();
	wordPos := tick.MezzKey()

	// 	return keccak256(abi.encodePacked(poolIdx, wordPos));
	tmpWordPos := make([]byte, 2)
	binary.LittleEndian.PutUint16(tmpWordPos, uint16(wordPos))

	h := sha3.NewLegacyKeccak256()
	h.Write(abi.EncodePacked([]byte(poolIdx), tmpWordPos))

	return string(h.Sum(nil))
}

/* @notice Formats the tick bit horizon index and sets the flag for whether it
 *          represents whether the seeks spills over the terminus neighborhood */
func pinTermMezz(isUpper bool, shiftTerm uint16, tickMezz int16, termBitMap *big.Int) (Int24, bool, error) {
	// (uint8 nextTerm, bool spillTrunc) =
	// termBitmap.bitAfterTrunc(shiftTerm, isUpper);
	nextTerm, spillTrunc, err := bitAfterTrunc(termBitMap, shiftTerm, isUpper)
	if err != nil {
		return 0, false, nil
	}

	// spillBit = doesSpillBit(isUpper, spillTrunc, termBitmap);
	spillBit, err := doesSpillBit(isUpper, spillTrunc, termBitMap)
	if err != nil {
		return 0, false, err
	}

	// nextTick = spillBit ?
	// spillOverPin(isUpper, tickMezz) :
	// Bitmaps.weldMezzTerm(tickMezz, nextTerm);
	var nextTick Int24
	if spillBit {
		nextTick = spillOverPin(isUpper, tickMezz)
	} else {
		nextTick = weldMezzTerm(tickMezz, nextTerm)
	}

	return nextTick, spillBit, nil
}

//     function pinTermMezz (bool isUpper, uint16 shiftTerm, int16 tickMezz,
// 		uint256 termBitmap)
// private pure returns (int24 nextTick, bool spillBit) {

// }

/* @notice Returns true if the tick seek reaches the end of the inner terminus
*      bitmap neighborhood. If that happens, it's like reaching the end of the map.
*      It's returned as the boundary point, but the the user must be aware that the tick
*      may or may not represent an active liquidity tick and check accordingly. */
func doesSpillBit(isUpper bool, spillTrunc bool, termBitMap *big.Int) (bool, error) {
	// 	 if (isUpper) {
	// 		 spillBit = spillTrunc;
	// 	 }
	var spillBit bool
	if isUpper {
		spillBit = spillTrunc
		return spillBit, nil
	}

	// 		 bool bumpAtFloor = termBitmap.isBitSet(0);
	bumpAtFloor, err := isBitSet(termBitMap, 0)
	if err != nil {
		return false, nil
	}

	// 		 spillBit = bumpAtFloor ? false :
	// 			 spillTrunc;
	if bumpAtFloor {
		spillBit = false
		return spillBit, nil
	}
	spillBit = spillTrunc
	return spillBit, nil
}

/* @notice Formats the censored horizon tick index when the seek has spilled out of
*         the terminus bitmap neighborhood. */
//  function spillOverPin (bool isUpper, int16 tickMezz) private pure returns (int24) {
func spillOverPin(isUpper bool, tickMezz int16) Int24 {
	//     if (isUpper) {
	if isUpper {
		//         return tickMezz == Bitmaps.zeroMezz(isUpper) ?
		//             Bitmaps.zeroTick(isUpper) :
		//             Bitmaps.weldMezzTerm(tickMezz + 1, Bitmaps.zeroTerm(!isUpper));
		if tickMezz == zeroMezz(isUpper) {
			return zeroTick(isUpper)
		}
		return weldMezzTerm(tickMezz+1, zeroTerm(!isUpper))
	}

	//         return Bitmaps.weldMezzTerm(tickMezz, 0);
	return weldMezzTerm(tickMezz, 0)
}

/* @notice Unset the tick index as no longer active. Take care of any book keeping
*   related to the recursive bitmap levels.
* @dev Idempontent. Can be called repeatedly even if tick was previously
*   forgotten.
* @param poolIdx The hash key associated with the pool being queried.
* @param tick The price tick that we're marking as disabled. */
func forgetTick(poolIdx string, tick Int24) {
	//     uint256 mezzMask = ~(1 << tick.mezzBit());
	mezzMask := big.NewInt(1)
	mezzMask.Lsh(mezzMask, uint(tick.MezzBit()))
	mezzMask.Not(mezzMask)

	// uint256 termMask = ~(1 << tick.termBit());
	termMask := big.NewInt(1)
	termMask.Lsh(termMask, uint(termBit(tick)))
	termMask.Not(termMask)

	//     bytes32 termIdx = encodeTerm(poolIdx, tick);
	termIdx := encodeTerm(poolIdx, tick)

	//     uint256 termUpdate = terminus_[termIdx] & termMask;
	termUpdate := terminus(termIdx)
	termUpdate.And(termUpdate, termMask)

	// terminus_[termIdx] = termUpdate;
	setTerminus(termIdx, termUpdate)

	//     if (termUpdate == 0) {
	//         bytes32 mezzIdx = encodeMezz(poolIdx, tick);
	//         uint256 mezzUpdate = mezzanine_[mezzIdx] & mezzMask;
	//         mezzanine_[mezzIdx] = mezzUpdate;
	//     }
	if termUpdate.Cmp(big0) == 0 {
		mezzIdx := encodeMezz(poolIdx, tick)
		mezzUpdate := mezzanine(mezzIdx)
		mezzUpdate.And(mezzUpdate, mezzMask)
		setMezzanine(mezzIdx, mezzUpdate)
	}
}

/* @notice Encodes the hash key for the mezzanine neighborhood of the tick. */
func encodeMezz(poolIdx string, tick Int24) string {
	wordPos := lobbyKey(tick)

	h := sha3.NewLegacyKeccak256()
	h.Write(abi.EncodePacked([]byte(poolIdx), []byte{byte(wordPos)}))

	return string(h.Sum(nil))
}

// function encodeMezz (bytes32 poolIdx, int24 tick) private pure returns (bytes32) {
//     int8 wordPos = tick.lobbyKey();
//     return keccak256(abi.encodePacked(poolIdx, wordPos));
// }

// TODO: how to get the terminus map???
func terminus(_ string) *big.Int {
	return nil
}

// TODO: how to set the terminus map???
func setTerminus(_ string, _ *big.Int) {
	return
}

// TODO: how to get the mezzanine map???
func mezzanine(_ string) *big.Int {
	return nil
}

// TODO: how to set the mezzanine map???
func setMezzanine(_ string, _ *big.Int) {
	return
}

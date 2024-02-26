package ambient

import (
	"encoding/binary"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient/types"
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
func seekMezzSpill(poolIdx string, borderTick types.Int24, isUpper bool) types.Int24 {
	// 	 if (isUpper && borderTick == type(int24).max) { return type(int24).max; }
	if isUpper && borderTick == types.Int24Max {
		return types.Int24Max
	}

	// 	 if (!isUpper && borderTick == type(int24).min) { return type(int24).min; }
	if !isUpper && borderTick == types.Int24Min {
		return types.Int24Min
	}

	// 	 (uint8 lobbyBorder, uint8 mezzBorder) = rootsForBorder(borderTick, isUpper);
	lobbyBorder, mezzBorder := rootsForBorder(borderTick, isUpper)

	// 	 // Most common case is that the next neighboring bitmap on the border has
	// 	 // an active tick. So first check here to save gas in the hotpath.
	// 	 (int24 pin, bool spills) =
	// 		 seekAtTerm(poolIdx, lobbyBorder, mezzBorder, isUpper);

	// 	 if (!spills) { return pin; }

}

/* @notice Seeks the next tick bitmap by searching in the adjacent neighborhood. */
func seekAtTerm(poolIdx string, lobbyBit uint8, mezzBit uint8, isUpper bool) (types.Int24, bool) {
	//     uint256 neighborBitmap = terminus_
	//         [encodeTermWord(poolIdx, lobbyBit, mezzBit)];
	neighborBitmap := terminus(encodeTermWord(poolIdx, lobbyBit, mezzBit))

	// (uint8 termBit, bool spills) = neighborBitmap.bitAfterTrunc(0, isUpper);
	termBit, spills := 

}

// function seekAtTerm (bytes32 poolIdx, uint8 lobbyBit, uint8 mezzBit, bool isUpper)
//     private view returns (int24, bool) {

//     if (spills) { return (0, true); }
//     return (Bitmaps.weldLobbyPosMezzTerm(lobbyBit, mezzBit, termBit), false);
// }

// 	 // Next check to see if we can find a neighbor in the mezzanine. This almost
// 	 // always happens except for very sparse pools.
// 	 (pin, spills) =
// 		 seekAtMezz(poolIdx, lobbyBorder, mezzBorder, isUpper);
// 	 if (!spills) { return pin; }

// 	 // Finally iterate through the lobby layer.
// 	 return seekOverLobby(poolIdx, lobbyBorder, isUpper);
//  }

/* @notice Encodes the hash key for the terminus neighborhood of the first 16-bits
 *         of a tick index. (This is all that's needed to determine terminus.) */
func encodeTermWord(poolIdx string, lobbyPos uint8, mezzPos uint8) string {
	mezzIdx := weldLobbyMezz(uncastBitmapIndex(lobbyPos), mezzPos)

	h := sha3.NewLegacyKeccak256()
	tmp := make([]byte, 2)
	binary.LittleEndian.PutUint16(tmp, uint16(mezzIdx))
	h.Write(abi.EncodePacked([]byte(poolIdx), tmp))

	return string(h.Sum(nil))
}

// 	 function encodeTermWord (bytes32 poolIdx, uint8 lobbyPos, uint8 mezzPos)
// 	 private pure returns (bytes32) {
// 	 int16 mezzIdx = Bitmaps.weldLobbyMezz
// 		 (Bitmaps.uncastBitmapIndex(lobbyPos), mezzPos);
// 	 return keccak256(abi.encodePacked(poolIdx, mezzIdx));
//  }

/* @notice Splits out the lobby bits and the mezzanine bits from the 24-bit price
*         tick index associated with the type of border tick used in seekMezzSpill()
*         call */
func rootsForBorder(borderTick types.Int24, isUpper bool) (uint8, uint8) {
	// Because pinTermMezz returns a border *on* the previous bitmap, we need to
	// decrement by one to get the seek starting point.
	// int24 pinTick = isUpper ? borderTick : (borderTick - 1);
	var pinTick types.Int24
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

// TODO: how to get the terminus map???
func terminus(_ string) *big.Int {
	return nil
}

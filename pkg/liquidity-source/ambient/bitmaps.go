package ambient

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient/types"
)

/* @notice Converts an 8-bit lobby index, an 8-bit mezzanine bit, and an 8-bit
 *   terminus bit into a full 24-bit tick index. */
func weldLobbyMezzTerm(lobbyIdx int8, mezzBitArg uint8, termBitArg uint8) types.Int24 {
	// First term will always be  <= 0x8F0000. Second term, starting as a uint8
	// will always be positive and <= 0xFF00. Thir term will always be positive
	// and <= 0xFF. Therefore the sum will never overflow int24
	// 	 return (int24(lobbyIdx) << 16) +
	// 		 (int24(uint24(mezzBitArg)) << 8) +
	// 		 int24(uint24(termBitArg));
	// 	 }
	return (types.Int24(lobbyIdx) << 16) +
		(types.Int24(types.Uint24(mezzBitArg)) << 8) +
		types.Int24(types.Uint24(termBitArg))
}

/* @notice Converts an 8-bit lobby index and an 8-bit mezzanine bit into a 16-bit
 *   tick base root. */
func weldLobbyMezz(lobbyIdx int8, mezzBitArg uint8) int16 {
	// First term will always be <= 0x8F00 and second term (as a uint) will always
	// be positive and <= 0xFF. Therefore the sum will never overflow int24
	// 	 return (int16(lobbyIdx) << 8) + int16(uint16(mezzBitArg));
	// 	 }
	return (int16(lobbyIdx) << 8) + int16(uint16(mezzBitArg))
}

/* @notice Converts an unsigned integer bitmap index to a signed integer. */
func uncastBitmapIndex(x uint8) int8 {
	//     return x < 128 ?
	//         int8(int16(uint16(x)) - 128) : // max(uint8) - 128 <= 127, so never overflows int8
	//         int8(x - 128);  // min(uint8) - 128  >= -128, so never underflows int8
	//     }
	if x < 128 {
		return int8(int16(uint16(x)) - 128)
	}
	return int8(x - 128)
}

/* @notice - Determine the index of the first set bit in the bitmap starting
*    after N bits from the right or the left.
* @param bitmap - The 256-bit bitmap object.
* @param shift - Exclude the first shift N bits from the index result.
* @param right - If true find the first set bit starting from the right
*   (least significant bit as EVM is big endian). Otherwise from the lefft.
* @return idx - The index of the matching set bit. Index position is always
*   left indexed starting at zero regardless of the @right parameter.
* @return spills - If no matching set bit is found, this return value is set to
*   true. */
func bitAfterTrunc(bitmap *big.Int, shift uint16, right bool) (uint8, bool) {
	bitmap = truncateBitmap(bitmap, shift, right)
	spills := bitmap.Cmp(big0) == 0

	if !spills {
		if right {

		}
	}
}

// 	 function bitAfterTrunc (uint256 bitmap, uint16 shift, bool right)
// 	 pure internal returns (uint8 idx, bool spills) {
// 	 bitmap = truncateBitmap(bitmap, shift, right);
// 	 spills = (bitmap == 0);
// 	 if (!spills) {
// 		 idx = right ?
// 			 BitMath.leastSignificantBit(bitmap) :
// 			 BitMath.mostSignificantBit(bitmap);
// 	 }
//  }

/* @notice Transforms the bitmap so the first or last N bits are set to zero.
* @param bitmap - The original 256-bit bitmap object.
* @param shift - The number N of slots in the bitmap to mask to zero.
* @param right - If true mask the N bits from right to left. Otherwise from
*                left to right.
* @return The bitmap with N bits (on the right or left side) masked. */
func truncateBitmap(bitmap *big.Int, shift uint16, right bool) *big.Int {
	// return right ?
	// 	 (bitmap >> shift) << shift:
	// 	 (bitmap << shift) >> shift;

	res := new(big.Int).Set(bitmap)
	if right {
		res.Rsh(res, uint(shift))
		res.Lsh(res, uint(shift))
		return res
	}
	res.Lsh(res, uint(shift))
	res.Rsh(res, uint(shift))
	return res
}

/// @notice Returns the index of the most significant bit of the number,
    ///     where the least significant bit is at index 0 and the most significant bit is at index 255
    /// @dev The function satisfies the property:
    ///     x >= 2**mostSignificantBit(x) and x < 2**(mostSignificantBit(x)+1)
    /// @param x the value for which to compute the most significant bit, must be greater than 0
    /// @return r the index of the most significant bit
    function mostSignificantBit(uint256 x) internal pure returns (uint8 r) {
        // Set to unchecked, but the original UniV3 library was written in a pre-checked version of Solidity
        unchecked{
        require(x > 0);

        if (x >= 0x100000000000000000000000000000000) {
            x >>= 128;
            r += 128;
        }
        if (x >= 0x10000000000000000) {
            x >>= 64;
            r += 64;
        }
        if (x >= 0x100000000) {
            x >>= 32;
            r += 32;
        }
        if (x >= 0x10000) {
            x >>= 16;
            r += 16;
        }
        if (x >= 0x100) {
            x >>= 8;
            r += 8;
        }
        if (x >= 0x10) {
            x >>= 4;
            r += 4;
        }
        if (x >= 0x4) {
            x >>= 2;
            r += 2;
        }
        if (x >= 0x2) r += 1;
        }
    }

    /// @notice Returns the index of the least significant bit of the number,
    ///     where the least significant bit is at index 0 and the most significant bit is at index 255
    /// @dev The function satisfies the property:
    ///     (x & 2**leastSignificantBit(x)) != 0 and (x & (2**(leastSignificantBit(x)) - 1)) == 0)
    /// @param x the value for which to compute the least significant bit, must be greater than 0
    /// @return r the index of the least significant bit
    function leastSignificantBit(uint256 x) internal pure returns (uint8 r) {
        // Set to unchecked, but the original UniV3 library was written in a pre-checked version of Solidity
        unchecked {
        require(x > 0);

        r = 255;
        if (x & type(uint128).max > 0) {
            r -= 128;
        } else {
            x >>= 128;
        }
        if (x & type(uint64).max > 0) {
            r -= 64;
        } else {
            x >>= 64;
        }
        if (x & type(uint32).max > 0) {
            r -= 32;
        } else {
            x >>= 32;
        }
        if (x & type(uint16).max > 0) {
            r -= 16;
        } else {
            x >>= 16;
        }
        if (x & type(uint8).max > 0) {
            r -= 8;
        } else {
            x >>= 8;
        }
        if (x & 0xf > 0) {
            r -= 4;
        } else {
            x >>= 4;
        }
        if (x & 0x3 > 0) {
            r -= 2;
        } else {
            x >>= 2;
        }
        if (x & 0x1 > 0) r -= 1;
        }
    }

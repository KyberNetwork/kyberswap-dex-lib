package ambient

import (
	"errors"
	"math"
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
func bitAfterTrunc(bitmap *big.Int, shift uint16, right bool) (uint8, bool, error) {
	// 	 bitmap = truncateBitmap(bitmap, shift, right);
	bitmap = truncateBitmap(bitmap, shift, right)

	// 	 spills = (bitmap == 0);
	spills := bitmap.Cmp(big0) == 0

	var (
		idx uint8
		err error
	)

	// 	 if (!spills) {
	// 		 idx = right ?
	// 			 BitMath.leastSignificantBit(bitmap) :
	// 			 BitMath.mostSignificantBit(bitmap);
	// 	 }
	if !spills {
		if right {
			idx, err = leastSignificantBit(bitmap)
			if err != nil {
				return 0, false, err
			}
		} else {
			idx, err = mostSignificantBit(bitmap)
			if err != nil {
				return 0, false, err
			}
		}
	}
	return idx, spills, nil
}

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

// / @notice Returns the index of the most significant bit of the number,
// /     where the least significant bit is at index 0 and the most significant bit is at index 255
// / @dev The function satisfies the property:
// /     x >= 2**mostSignificantBit(x) and x < 2**(mostSignificantBit(x)+1)
// / @param x the value for which to compute the most significant bit, must be greater than 0
// / @return r the index of the most significant bit
var (
	big0x100000000000000000000000000000000, _ = new(big.Int).SetString("0x100000000000000000000000000000000", 16)
	big0x10000000000000000, _                 = new(big.Int).SetString("0x10000000000000000", 16)
	big0x100000000, _                         = new(big.Int).SetString("0x100000000", 16)
	big0x10000, _                             = new(big.Int).SetString("0x10000", 16)
	big0x100, _                               = new(big.Int).SetString("0x100", 16)
	big0x10, _                                = new(big.Int).SetString("0x10", 16)
	big0x4, _                                 = new(big.Int).SetString("0x4", 16)
	big0x2, _                                 = new(big.Int).SetString("0x2", 16)
)

func mostSignificantBit(x *big.Int) (uint8, error) {
	//     require(x > 0);
	if x.Cmp(big0) < 1 {
		return 0, errors.New("x need to be larger than 0")
	}

	var r uint8

	//     if (x >= 0x100000000000000000000000000000000) {
	//         x >>= 128;
	//         r += 128;
	//     }
	if x.Cmp(big0x100000000000000000000000000000000) > -1 {
		x.Rsh(x, 128)
		r += 128
	}

	//     if (x >= 0x10000000000000000) {
	//         x >>= 64;
	//         r += 64;
	//     }
	if x.Cmp(big0x10000000000000000) > -1 {
		x.Rsh(x, 64)
		r += 64
	}

	//     if (x >= 0x100000000) {
	//         x >>= 32;
	//         r += 32;
	//     }
	if x.Cmp(big0x100000000) > -1 {
		x.Rsh(x, 32)
		r += 32
	}

	//     if (x >= 0x10000) {
	//         x >>= 16;
	//         r += 16;
	//     }
	if x.Cmp(big0x10000) > -1 {
		x.Rsh(x, 16)
		r += 16
	}

	//     if (x >= 0x100) {
	//         x >>= 8;
	//         r += 8;
	//     }
	if x.Cmp(big0x100) > -1 {
		x.Rsh(x, 8)
		r += 8
	}

	//     if (x >= 0x10) {
	//         x >>= 4;
	//         r += 4;
	//     }
	if x.Cmp(big0x10) > -1 {
		x.Rsh(x, 4)
		r += 4
	}

	//     if (x >= 0x4) {
	//         x >>= 2;
	//         r += 2;
	//     }
	if x.Cmp(big0x4) > -1 {
		x.Rsh(x, 2)
		r += 2
	}

	//     if (x >= 0x2) r += 1;
	if x.Cmp(big0x2) > -1 {
		r += 1
	}

	return r, nil
}

// / @notice Returns the index of the least significant bit of the number,
// /     where the least significant bit is at index 0 and the most significant bit is at index 255
// / @dev The function satisfies the property:
// /     (x & 2**leastSignificantBit(x)) != 0 and (x & (2**(leastSignificantBit(x)) - 1)) == 0)
// / @param x the value for which to compute the least significant bit, must be greater than 0
// / @return r the index of the least significant bit
var (
	big0xf, _ = new(big.Int).SetString("0xf", 16)
	big0x3, _ = new(big.Int).SetString("0x3", 16)
	big0x1, _ = new(big.Int).SetString("0x1", 16)
)

func leastSignificantBit(inX *big.Int) (uint8, error) {
	x := new(big.Int).Set(inX)

	//     require(x > 0);
	if x.Cmp(big0) < 1 {
		return 0, errors.New("x need to be larger than 0")
	}
	var (
		//     r = 255;
		r   uint8 = 255
		tmp       = new(big.Int)
	)

	//     if (x & type(uint128).max > 0) {
	//         r -= 128;
	//     } else {
	//         x >>= 128;
	//     }
	if tmp.And(x, bigMaxUint128).Cmp(big0) == 1 {
		r -= 128
	} else {
		x.Rsh(x, 128)
	}

	//     if (x & type(uint64).max > 0) {
	//         r -= 64;
	//     } else {
	//         x >>= 64;
	//     }
	if tmp.And(x, bigMaxUint64).Cmp(big0) == 1 {
		r -= 64
	} else {
		x.Rsh(x, 64)
	}

	//     if (x & type(uint32).max > 0) {
	//         r -= 32;
	//     } else {
	//         x >>= 32;
	//     }
	if tmp.And(x, bigMaxUint32).Cmp(big0) == 1 {
		r -= 32
	} else {
		x.Rsh(x, 32)
	}

	//     if (x & type(uint16).max > 0) {
	//         r -= 16;
	//     } else {
	//         x >>= 16;
	//     }
	if tmp.And(x, bigMaxUint16).Cmp(big0) == 1 {
		r -= 16
	} else {
		x.Rsh(x, 16)
	}

	//     if (x & type(uint8).max > 0) {
	//         r -= 8;
	//     } else {
	//         x >>= 8;
	//     }
	if tmp.And(x, bigMaxUint8).Cmp(big0) == 1 {
		r -= 8
	} else {
		x.Rsh(x, 8)
	}

	//     if (x & 0xf > 0) {
	//         r -= 4;
	//     } else {
	//         x >>= 4;
	//     }
	if tmp.And(x, big0xf).Cmp(big0) == 1 {
		r -= 4
	} else {
		x.Rsh(x, 4)
	}

	//     if (x & 0x3 > 0) {
	//         r -= 2;
	//     } else {
	//         x >>= 2;
	//     }
	if tmp.And(x, big0x3).Cmp(big0) == 1 {
		r -= 2
	} else {
		x.Rsh(x, 2)
	}

	//     if (x & 0x1 > 0) r -= 1;
	if tmp.And(x, big0x1).Cmp(big0) == 1 {
		r -= 1
	}

	return r, nil
}

/* @notice Converts a directional bitmap position, to a cardinal bitmap position. For
*   example the 20th bit for a sell (right-to-left) would be the 235th bit in
*   the bitmap.
* @param bit - The directional-oriented index in the 256-bit bitmap.
* @param isUpper - If true, the direction is left-to-right, if false right-to-left.
* @return The cardinal (left-to-right) index in the bitmap. */
func bitRelate(bit uint8, isUpper bool) uint8 {
	// return isUpper ? bit : (255 - bit); // 255 minus uint8 will never underflow
	if isUpper {
		return bit
	}
	return 255 - bit
}

/* @notice Returns the zero horizon point for the full 24-bit tick index. */
func zeroTick(isUpper bool) types.Int24 {
	//     return isUpper ? type(int24).max : type(int24).min;
	if isUpper {
		return types.Int24Max
	}
	return types.Int24Min
}

/* @notice The minimum and maximum 24-bit integers are used to represent -/+
*   infinity range. We have to reserve these bits as non-standard range for when
*   price shifts past the last representable tick.
* @param tick The tick index value being tested
* @return True if the tick index represents a positive or negative infinity. */
func isTickFinite(tick types.Int24) bool {
	//     return tick > type(int24).min &&
	//         tick < type(int24).max;
	return tick > types.Int24Min && tick < types.Int24Max
}

/* @notice Determines the next shift bump from a starting terminus value. Note for
*   upper the barrier is always to the right. For lower it's on the tick. This is
*   because bumps always occur at the start of the tick.
*
* @param tick - The full 24-bit tick index.
* @param isUpper - If true, shift and index from left-to-right. Otherwise right-to-
*   left.
* @return - Returns the bumped terminus bit indexed directionally based on param
*   isUpper. Can be 256, if the terminus bit occurs at the last slot. */
func termBump(tick types.Int24, isUpper bool) uint16 {
	return 0
}

// 	function termBump (int24 tick, bool isUpper) internal pure returns (uint16) {
// 	unchecked {
// 	uint8 bit = termBit(tick);
// 	// Bump moves up for upper, but occurs at the bottom of the same tick for lower.
// 	uint16 shiftTerm = isUpper ? 1 : 0;
// 	return uint16(bitRelate(bit, isUpper)) + shiftTerm;
// 	}
// }

/* @notice Extracts the 8-bit terminus bits (the last 8-bits) from the full 24-bit
* tick index. Result can be used to index on a terminus bitmap. */
func termBit(tick types.Int24) uint8 {
	//     return uint8(uint24(tick % 256)); // Modulo 256 will always <= 255, and fit in uint8
	return uint8(types.Uint24(tick % 256))
}

/* @notice Returns true if the bitmap's Nth bit slot is set.
 * @param bitmap - The 256 bit bitmap object.
 * @param pos - The bitmap index to check. Value is left indexed starting at zero.
 * @return True if the bit is set. */
func isBitSet(bitmap *big.Int, pos uint8) (bool, error) {
	//     (uint idx, bool spill) = bitAfterTrunc(bitmap, pos, true);
	idx, spill, err := bitAfterTrunc(bitmap, uint16(pos), true)
	if err != nil {
		return false, err
	}

	//     return !spill && idx == pos;
	return !spill && idx == pos, nil
}

/* @notice Returns the zero horizon point equivalent for the first 16-bits of the
*    tick index. */
func zeroMezz(isUpper bool) int16 {
	//     return isUpper ? type(int16).max : type(int16).min;
	if isUpper {
		return math.MaxInt16
	}
	return math.MinInt16
}

/* @notice Returns the zero point equivalent for the terminus bit (last 8-bits) of
*    the tick index. */
func zeroTerm(isUpper bool) uint8 {
	//     return isUpper ? type(uint8).max : 0;
	if isUpper {
		return math.MaxUint8
	}

	return 0
}

/* @notice Converts a 16-bit tick base and an 8-bit terminus tick to a full 24-bit
*   tick index. */
func weldMezzTerm(mezzBase int16, termBitArg uint8) types.Int24 {
	// 	 // First term will always be <= 0x8FFF00 and second term (as a uint8) will always
	// 	 // be positive and <= 0xFF. Therefore the sum will never overflow int24
	// 	 return (int24(mezzBase) << 8) + int24(uint24(termBitArg));
	return (types.Int24(mezzBase) << 8) + types.Int24(types.Uint24(termBitArg))
}

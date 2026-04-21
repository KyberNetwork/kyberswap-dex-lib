package ambient

import (
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var mask256 = bignum.MaxUint256

func CastBitmapIndex(x int8) uint8 {
	if x >= 0 {
		return uint8(x) + 128
	}
	return uint8(int16(x) + 128)
}

func UncastBitmapIndex(x uint8) int8 {
	if x < 128 {
		return int8(int16(x) - 128)
	}
	return int8(x - 128)
}

func LobbyBit(tick int32) uint8 {
	return CastBitmapIndex(LobbyKey(tick))
}

func WeldMezzTerm(mezzBase int16, termBitArg uint8) int32 {
	return (int32(mezzBase) << 8) + int32(termBitArg)
}

func WeldLobbyMezz(lobbyIdx int8, mezzBitArg uint8) int16 {
	return (int16(lobbyIdx) << 8) + int16(mezzBitArg)
}

func WeldLobbyMezzTerm(lobbyIdx int8, mezzBitArg, termBitArg uint8) int32 {
	return (int32(lobbyIdx) << 16) + (int32(mezzBitArg) << 8) + int32(termBitArg)
}

// TruncateBitmap mirrors Bitmaps.truncateBitmap, operating on a 256-bit value.
// For right=true  → (bitmap >> shift) << shift  (zeroes the low shift bits).
// For right=false → (bitmap << shift) >> shift  (zeroes the high shift bits,
//
//	which in Solidity is implicit modulo-2²⁵⁶ overflow).
func TruncateBitmap(bitmap *big.Int, shift uint, right bool) *big.Int {
	result := new(big.Int).Set(bitmap)
	if right {
		result.Rsh(result, shift)
		result.Lsh(result, shift)
		result.And(result, mask256)
	} else {
		result.Lsh(result, shift)
		result.And(result, mask256) // mimic uint256 overflow wrap
		result.Rsh(result, shift)
	}
	return result
}

// IsBitSet mirrors Bitmaps.isBitSet — position 0 is LSB.
func IsBitSet(bitmap *big.Int, pos uint8) bool {
	return bitmap.Bit(int(pos)) == 1
}

func BitAfterTrunc(bitmap *big.Int, shift uint, right bool) (uint8, bool) {
	bm := TruncateBitmap(bitmap, shift, right)
	if bm.Sign() == 0 {
		return 0, true
	}
	if right {
		return uint8(bm.TrailingZeroBits()), false
	}
	return uint8(bm.BitLen() - 1), false
}

func TermBump(tick int32, isUpper bool) uint16 {
	bit := TermBit(tick)
	var shiftTerm uint16
	if isUpper {
		shiftTerm = 1
	}
	return uint16(bitRelate(bit, isUpper)) + shiftTerm
}

func bitRelate(bit uint8, isUpper bool) uint8 {
	if isUpper {
		return bit
	}
	return 255 - bit
}

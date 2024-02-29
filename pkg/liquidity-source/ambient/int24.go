package ambient

type Int24 int32

type Uint24 uint32

var (
	Int24Max Int24 = 8388607
	Int24Min Int24 = -8388608
)

/* @notice Extracts the 16-bit tick root from the full 24-bit tick
 * index. */
func (i Int24) MezzKey() int16 {
	return int16(i >> 8)
}

/* @notice Extracts the 8-bit lobby bits (the last 8-bits) from the full 24-bit tick
 * index. Result can be used to index on a lobby bitmap. */
func (i Int24) LobbyBit() uint8 {
	//     return castBitmapIndex(lobbyKey(tick));
	return castBitmapIndex(lobbyKey(i))
}

/* @notice Extracts the 8-bit mezznine bits (the middle 8-bits) from the full 24-bit
 * tick index. Result can be used to index on a mezzanine bitmap. */
func (i Int24) MezzBit() uint8 {
	//     return uint8(uint16(mezzKey(tick) % 256)); // Modulo 256 will always <= 255, and fit in uint8
	return uint8(uint16(i.MezzKey() % 256))
}

/* @notice Converts a signed integer bitmap index to an unsigned integer. */
func castBitmapIndex(x int8) uint8 {
	// 	return x >= 0 ?
	// 		uint8(x) + 128 : // max(int8(x)) + 128 <= 255, so this never overflows
	// 		uint8(uint16(int16(x) + 128)); // min(int8(x)) + 128 >= 0 (and less than 255)
	// 	}

	if x >= 0 {
		return uint8(x) + 128
	}

	return uint8(uint16(int16(x) + 128))
}

/* @notice Extracts the 8-bit tick lobby index from the full 24-bit tick index. */
func lobbyKey(tick Int24) int8 {
	//     return int8(tick >> 16); // 24-bit int shifted by 16 bits will always fit in 8 bits
	return int8(tick >> 16)
}

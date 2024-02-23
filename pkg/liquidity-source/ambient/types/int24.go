package types

type Int24 int32

/* @notice Extracts the 16-bit tick root from the full 24-bit tick
 * index. */
func (i Int24) mezzKey(tick Int24) int16 {
	return int16(tick >> 8)
}
package helper

import "math/big"

const (
	ZX = "0x"
)

func newBitMask(start uint, end uint) *big.Int {
	mask := big.NewInt(1)
	mask.Lsh(mask, end)
	mask.Sub(mask, big.NewInt(1))
	if start == 0 {
		return mask
	}

	notMask := newBitMask(0, start)
	notMask.Not(notMask)
	mask.And(mask, notMask)

	return mask
}

func setMask(n *big.Int, mask *big.Int, value *big.Int) {
	// Clear bits in range.
	n.And(n, new(big.Int).Not(mask))

	// Shift value to correct position and ensure value fits in mask.
	value = new(big.Int).Lsh(value, mask.TrailingZeroBits())
	value.And(value, mask)

	// Set the bits in range.
	n.Or(n, value)
}

func getMask(n *big.Int, start, end uint) *big.Int {
	mask := new(big.Int).Lsh(big.NewInt(1), end)
	mask.Sub(mask, new(big.Int).Lsh(big.NewInt(1), start))
	result := new(big.Int).And(n, mask)
	return result.Rsh(result, start)
}

package math

// SpreadDenominator is SpreadMath.DENOMINATOR: one whole reference price. Spreads and offsets are
// millionths of it, so 1 unit is 0.01 bps.
const SpreadDenominator = 1_000_000

// Spread is the pool's Spread: an unsigned distance from a reference price, in millionths.
type Spread uint32

// SignedSpread is the pool's SignedSpread: a signed offset from a reference price, in millionths.
type SignedSpread int32

// Add is SpreadMath.add. A sum above uint32 is ErrOverflow, as Solidity's checked uint32 addition
// reverts.
func (s Spread) Add(other Spread) (Spread, error) {
	sum := uint64(s) + uint64(other)
	if sum > uint64(^uint32(0)) {
		return 0, ErrOverflow
	}
	return Spread(sum), nil
}

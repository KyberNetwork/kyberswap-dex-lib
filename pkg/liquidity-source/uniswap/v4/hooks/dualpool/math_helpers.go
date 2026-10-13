package dualpool

import "github.com/holiman/uint256"

// Position exposes an allocated range for derived hooks with their own settlement.
type Position struct {
	TickLower, TickUpper            int
	SqrtLower, SqrtUpper, Liquidity *uint256.Int
}

// AllocatePositions reuses DualPool's bucket allocation without changing its quote path.
func AllocatePositions(buckets []Bucket, price, balance0, balance1 *uint256.Int) ([]Position, error) {
	positions, err := allocate(buckets, price, balance0, balance1)
	if err != nil {
		return nil, err
	}
	result := make([]Position, len(positions))
	for i, p := range positions {
		result[i] = Position{p.tickLower, p.tickUpper, p.sqrtLower, p.sqrtUpper, p.liquidity}
	}
	return result, nil
}

// ComputeSwapStepExactIn exposes the existing V4 step math to derived hooks.
func ComputeSwapStepExactIn(current, target, liquidity, remaining *uint256.Int, fee uint64) (*uint256.Int, *uint256.Int, *uint256.Int, *uint256.Int, error) {
	return computeSwapStepExactIn(current, target, liquidity, remaining, fee)
}

// EffectiveSwapFee returns the existing direction-specific combined swap fee.
func EffectiveSwapFee(lp, protocol uint32, zeroForOne bool) uint64 {
	return effectiveSwapFee(lp, protocol, zeroForOne)
}

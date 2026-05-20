package capricornpamm

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// QuoteAmountOut interpolates the depth ladder to produce a post-fee
// amountOut. The on-chain curve is concave-down, so linear interpolation
// between grid points always slightly underestimates — quote slippage on
// the executed swap is non-negative.
//
// Contract: ladder is sorted ascending by AmountIn, all AmountIn > 0, all
// entries are non-zero quotes (tracker filters reverts and zeros).
//
// Rules:
//   - amountIn ≤ ladder[0].AmountIn  → scale linearly from origin
//   - amountIn between two grid points → linear interp
//   - amountIn equals a grid point → that point's AmountOut
//   - amountIn > largest grid point → ErrAmountInTooLarge (mirrors on-chain cap)
func QuoteAmountOut(ladder []LadderPoint, amountIn *uint256.Int) (*uint256.Int, error) {
	if amountIn == nil || amountIn.IsZero() {
		return nil, ErrZeroAmount
	}
	if len(ladder) == 0 {
		return nil, ErrNoQuote
	}

	// Below smallest grid point: scale from origin. amountIn ≤ first.AmountIn
	// ⇒ result ≤ first.AmountOut, so no overflow.
	first := ladder[0]
	if amountIn.Cmp(first.AmountIn) <= 0 {
		return big256.MulDiv(amountIn, first.AmountOut, first.AmountIn), nil
	}

	// Find bracket ladder[i].AmountIn < amountIn ≤ ladder[i+1].AmountIn.
	// The lower-endpoint case is caught by the previous iteration's exact-hit
	// branch (for i==0, by the cheap-path above).
	for i := 0; i < len(ladder)-1; i++ {
		lo, hi := ladder[i], ladder[i+1]
		if amountIn.Cmp(hi.AmountIn) == 0 {
			return new(uint256.Int).Set(hi.AmountOut), nil
		}
		if amountIn.Cmp(lo.AmountIn) > 0 && amountIn.Cmp(hi.AmountIn) < 0 {
			return interpolate(lo, hi, amountIn), nil
		}
	}
	return nil, ErrAmountInTooLarge
}

// interpolate computes lo.AmountOut + (amountIn-lo.AmountIn) * (hi.AmountOut-lo.AmountOut) / (hi.AmountIn-lo.AmountIn).
// Result is bounded above by hi.AmountOut. Floor rounding matches the
// on-chain quoter.
func interpolate(lo, hi LadderPoint, amountIn *uint256.Int) *uint256.Int {
	// Stack-allocated scratch; big256.MulDivDown treats inputs as read-only.
	var dxIn, rangeIn, rangeOut, delta uint256.Int
	dxIn.Sub(amountIn, lo.AmountIn)
	rangeIn.Sub(hi.AmountIn, lo.AmountIn)
	rangeOut.Sub(hi.AmountOut, lo.AmountOut)
	big256.MulDivDown(&delta, &dxIn, &rangeOut, &rangeIn)
	return new(uint256.Int).Add(lo.AmountOut, &delta)
}

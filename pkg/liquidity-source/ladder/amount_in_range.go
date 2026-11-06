package ladder

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// AmountInRange implements pool.IAmountInRange: it reports how much tokenIn this pool will quote
// without being asked to quote any particular size.
//
// A ladder pool is priced from a snapshot probed out to a finite amount. Inside that extent it
// quotes normally; one wei beyond it, QuoteAmountOut returns ErrAmountInTooLarge rather than
// extrapolating a curve it never measured. That makes the refusal a capacity limit and not a price
// signal, which a caller sampling at a fixed size cannot tell apart -- it sees only an error and
// must either discard the pool or invent a number for it. Reporting the extent lets the caller
// sample inside it instead.
//
// There is no minimum. NewSpline prepends zero to its knots, so every amountIn in (0, extent]
// resolves to a segment; minIn is therefore nil, meaning unbounded below, rather than the first
// probed rung.
//
// # What this bound does and does not include
//
// It is the ladder's extent only. CalcAmountOut can still refuse a size inside that extent for two
// reasons this deliberately ignores:
//
//   - the quote exceeding the output reserve (ErrInsufficientLiquidity), and
//   - the quote exceeding the caller's SwapLimit inventory (pool.ErrNotEnoughInventory).
//
// Both are bounds on amountOut, so folding either into a maxIn would mean inverting the spline.
// Doing that at rung granularity would *understate* the range, and understating is the one error
// the interface forbids: a caller detects an overstated bound by probing inside it and being
// refused, but an understated one silently shrinks what the caller will consider, with nothing to
// reveal it. So this overstates in those two cases on purpose, and leaves them to the probe.
//
// Consequently a successful AmountInRange is not a promise that maxIn quotes -- only that nothing
// above it can.
func (s *PoolSimulator) AmountInRange(tokenIn, tokenOut string, _ pool.SwapLimit) (*big.Int, *big.Int, error) {
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, nil, ErrInvalidToken
	}

	spline := s.splines[indexIn]
	if spline == nil || len(spline.x) <= 1 {
		// Nothing was probed in this direction, so there is no extent to report -- not even an
		// empty one. CalcAmountOut answers ErrNoQuote here. Report that the pool cannot say and
		// let the caller fall back to its own sampling, rather than let it read silence as a bound.
		return nil, nil, pool.ErrUnsupported
	}

	// Read the extent off the spline, not off Extra.Ladders. The spline's knots are the filtered
	// truth -- NewSpline drops any point whose amountIn is not strictly greater than the last kept
	// one -- while Extra.Ladders is the raw snapshot. The two diverge whenever a point is dropped:
	// a ladder probed past the pool's fill cap repeats its last point verbatim (harmless, equal
	// amountIn), but an out-of-order point leaves Extra.Ladders ending *below* the extent that
	// still quotes. Only the spline answers what QuoteAmountOut will accept.
	//
	// consumedIn carries earlier fills within this quoting round: CalcAmountOut evaluates the
	// spline at consumedIn+amountIn, so what remains is measured from there. Omitting it would
	// overstate a partly filled pool by everything already taken out of it.
	remaining := spline.x[len(spline.x)-1] - s.consumedIn[indexIn]
	if !(remaining > 0) {
		// Earlier fills consumed the whole ladder. Unlike the case above this is a statement, not
		// the absence of one, so it is reported as a zero maximum: the pool is alive and quotes
		// nothing, which spares the caller a probe that cannot succeed.
		return nil, new(big.Int), nil
	}

	// The knots hold wei as float64, so above ~9e15 their spacing already exceeds a wei and this
	// bound is approximate at that scale. Truncating toward zero keeps it on the quotable side of
	// the real extent.
	maxIn, _ := big.NewFloat(remaining).Int(nil)
	return nil, maxIn, nil
}
